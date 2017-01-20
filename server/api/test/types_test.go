package api

import (
	"encoding/json"

	"github.com/sdcoffey/olympus/server/api"
	. "gopkg.in/check.v1"
)

func (suite *ApiTestSuite) TestApiResponse_newErrorResponse_includesError(t *C) {
	response := api.NewErrorResponse(&api.ApiError{
		api.INTERNAL,
		"some details",
	})

	bytes, err := json.Marshal(&response)
	t.Check(err, IsNil)

	var unmarshaled map[string]interface{}
	err = json.Unmarshal(bytes, &unmarshaled)
	t.Check(err, IsNil)

	t.Check(unmarshaled["meta"], NotNil)
	meta := unmarshaled["meta"].(map[string]interface{})
	t.Check(meta["request_id"], NotNil)

	apiError := unmarshaled["error"].(map[string]interface{})
	t.Check(apiError, NotNil)
	t.Check(apiError["code"], Equals, "internal")
	t.Check(apiError["details"], Equals, "some details")

	t.Check(unmarshaled["data"], IsNil)
}

func (suite *ApiTestSuite) TestApiResponse_newDataResponse_includesData(t *C) {
	data := struct {
		One string `json:"one"`
		Two string `json:"two"`
	}{"foo", "bar"}
	response := api.NewDataResponse(data)

	bytes, err := json.Marshal(&response)
	t.Check(err, IsNil)

	var unmarshaled map[string]interface{}
	err = json.Unmarshal(bytes, &unmarshaled)
	t.Check(err, IsNil)

	t.Check(unmarshaled["meta"], NotNil)
	meta := unmarshaled["meta"].(map[string]interface{})
	t.Check(meta["request_id"], NotNil)

	unmarshaledData := unmarshaled["data"].(map[string]interface{})
	t.Check(unmarshaledData["one"], Equals, "foo")
	t.Check(unmarshaledData["two"], Equals, "bar")
}
