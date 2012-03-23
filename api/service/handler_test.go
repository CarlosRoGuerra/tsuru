package service_test

import (
	_ "github.com/mattn/go-sqlite3"
	. "github.com/timeredbull/tsuru/api/service"
	. "github.com/timeredbull/tsuru/api/app"
	. "launchpad.net/gocheck"
	"io/ioutil"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func Test(t *testing.T) { TestingT(t) }

type ServiceSuite struct {
	db          *sql.DB
	service     *Service
	serviceType *ServiceType
	serviceApp  *ServiceApp
}

var _ = Suite(&ServiceSuite{})

func (s *ServiceSuite) SetUpSuite(c *C) {
	s.db, _ = sql.Open("sqlite3", "./tsuru.db")

	_, err := s.db.Exec("CREATE TABLE 'service' ('id' INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, 'service_type_id' integer,'name' varchar(255))")
	c.Check(err, IsNil)

	_, err = s.db.Exec("CREATE TABLE 'service_type' ('id' INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, 'name' varchar(255), 'charm' varchar(255))")
	c.Check(err, IsNil)

	_, err = s.db.Exec("CREATE TABLE 'service_app' ('id' INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, 'service_id' integer, 'app_id' integer)")
	c.Check(err, IsNil)

	_, err = s.db.Exec("CREATE TABLE 'apps' ('id' INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, 'name' varchar(255), 'framework' varchar(255), 'state' varchar(255))")
	c.Check(err, IsNil)
}

func (s *ServiceSuite) TearDownSuite(c *C) {
	os.Remove("./tsuru.db")
	s.db.Close()
}

func (s *ServiceSuite) TearDownTest(c *C) {
	s.db.Exec("DELETE FROM service")
	s.db.Exec("DELETE FROM service_type")
	s.db.Exec("DELETE FROM service_app")
	s.db.Exec("DELETE FROM apps")
}

func (s *ServiceSuite) TestCreateHandler(c *C) {
	b := strings.NewReader(`{"name":"some_service", "type":"mysql"}`)
	request, err := http.NewRequest("POST", "/services", b)

	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	c.Assert(err, IsNil)

	CreateHandler(recorder, request)

	c.Assert(recorder.Body.String(), Equals, "success")
	c.Assert(recorder.Code, Equals, 200)

	rows, err := s.db.Query("SELECT count(*) FROM service WHERE name = 'some_service'")

	c.Check(err, IsNil)
	var qtd int

	for rows.Next() {
		rows.Scan(&qtd)
	}

	c.Assert(1, Equals, qtd)
}

func (s *ServiceSuite) TestServicesHandler(c *C) {
	st := ServiceType{Name: "Mysql", Charm: "mysql"}
	se := Service{ServiceTypeId: st.Id, Name: "myService"}
	se2 := Service{ServiceTypeId: st.Id, Name: "myOtherService"}
	st.Create()
	se.Create()
	se2.Create()

	request, err := http.NewRequest("GET", "/services", nil)
	c.Assert(err, IsNil)

	recorder := httptest.NewRecorder()
	ServicesHandler(recorder, request)
	c.Assert(recorder.Code, Equals, 200)

	body, err := ioutil.ReadAll(recorder.Body)
	c.Assert(err, IsNil)

	var results []ServiceT
	err = json.Unmarshal(body, &results)
	c.Assert(err, IsNil)
	c.Assert(len(results), Equals, 2)
	c.Assert(results[0], FitsTypeOf, ServiceT{})
	c.Assert(results[0].Id, Not(Equals), int64(0))
	c.Assert(results[0].Type, Not(Equals), "")
	c.Assert(results[0].Name, Not(Equals), "")
}

func (s *ServiceSuite) TestDeleteHandler(c *C) {
	se := Service{ServiceTypeId: 2, Name: "Mysql"}
	se.Create()
	request, err := http.NewRequest("GET", fmt.Sprintf("/services/%s?:name=%s", se.Name, se.Name), nil)
	c.Assert(err, IsNil)

	recorder := httptest.NewRecorder()
	DeleteHandler(recorder, request)
	c.Assert(recorder.Code, Equals, 200)

	rows, err := s.db.Query("SELECT count(*) FROM service WHERE name = 'Mysql'")
	c.Check(err, IsNil)

	var qtd int
	for rows.Next() {
		rows.Scan(&qtd)
	}

	c.Assert(qtd, Equals, 0)
}

func (s *ServiceSuite) TestDeleteHandlerReturns404(c *C) {
}

func (s *ServiceSuite) TestBindHandler(c *C) {
	st := ServiceType{Name: "Mysql", Charm: "mysql"}
	se := Service{ServiceTypeId: st.Id, Name: "my_service"}
	a := App{Name: "someApp", Framework: "django"}
	st.Create()
	se.Create()
	a.Create()

	b := strings.NewReader(`{"app":"someApp", "service":"my_service"}`)
	request, err := http.NewRequest("POST", "/services/bind", b)
	c.Assert(err, IsNil)

	recorder := httptest.NewRecorder()
	BindHandler(recorder, request)
	c.Assert(recorder.Code, Equals, 200)

	rows, err := s.db.Query("SELECT count(*) FROM service_app WHERE service_id = ? AND app_id = ?", se.Id, a.Id)
	c.Check(err, IsNil)

	var qtd int
	for rows.Next() {
		rows.Scan(&qtd)
	}

	c.Assert(qtd, Equals, 1)
}

func (s *ServiceSuite) TestUnbindHandler(c *C) {
	st := ServiceType{Name: "Mysql", Charm: "mysql"}
	se := Service{ServiceTypeId: st.Id, Name: "my_service"}
	a := App{Name: "someApp", Framework: "django"}
	st.Create()
	se.Create()
	a.Create()
	se.Bind(&a)

	b := strings.NewReader(`{"app":"someApp", "service":"my_service"}`)
	request, err := http.NewRequest("POST", "/services/bind", b)
	c.Assert(err, IsNil)

	recorder := httptest.NewRecorder()
	UnbindHandler(recorder, request)
	c.Assert(recorder.Code, Equals, 200)

	rows, err := s.db.Query("SELECT count(*) FROM service_app WHERE service_id = ? AND app_id = ?", se.Id, a.Id)
	c.Check(err, IsNil)

	var qtd int
	for rows.Next() {
		rows.Scan(&qtd)
	}

	c.Assert(qtd, Equals, 0)
}
