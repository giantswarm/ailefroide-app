package personio

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

const PERSONIO_API = "https://api.personio.de/v1"

type Personio struct {
	bearer    string
	ghFieldId string
}

type Employee struct {
	Email  string
	Github string
}

func New(clientid, secretid, ghFieldId string) (*Personio, error) {
	var p Personio = Personio{
		ghFieldId: ghFieldId,
	}
	var err error
	p.bearer, err = p.getBearer(clientid, secretid)
	return &p, err
}

func (p *Personio) Employees() (employees []Employee, err error) {
	var (
		request *http.Request
		data    struct {
			Success bool `json:"success"`
			Data    []struct {
				Type       string                            `json:"type"`
				Attributes map[string]map[string]interface{} `json:"attributes"`
			} `json:"data"`
		}
	)
	employees = make([]Employee, 0)

	request, err = http.NewRequest("GET", PERSONIO_API+"/company/employees", nil)
	if err != nil {
		return
	}

	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.bearer))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Add("Accept", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return
	}

	defer response.Body.Close()
	err = json.NewDecoder(response.Body).Decode(&data)
	if err != nil {
		return
	}

	for _, item := range data.Data {
		if item.Type == "Employee" {
			emp := Employee{
				Email:  item.Attributes["email"]["value"].(string),
				Github: item.Attributes[p.ghFieldId]["value"].(string),
			}
			employees = append(employees, emp)
		}
	}

	return
}

func (p *Personio) getBearer(clientid, secretid string) (bearer string, err error) {
	var (
		request *http.Request
		data    = struct {
			Clientid string `json:"client_id"`
			Secretid string `json:"client_secret"`
		}{
			Clientid: clientid,
			Secretid: secretid,
		}
		buffer *bytes.Buffer = new(bytes.Buffer)
	)

	json.NewEncoder(buffer).Encode(data)
	request, err = http.NewRequest("POST", PERSONIO_API+"/auth", buffer)
	if err != nil {
		return
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Add("Accept", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return
	}

	defer response.Body.Close()

	var body struct {
		Success bool `json:"success"`
		Data    struct {
			Token string `json:"token"`
		} `json:"data"`
	}

	json.NewDecoder(response.Body).Decode(&body)
	bearer = body.Data.Token
	return
}
