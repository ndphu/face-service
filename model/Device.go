package model

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
)

type Device struct {
	DeviceId string `json:"deviceId"`
}

func (d *Device) Capture() ([]byte, error) {
	if resp, err := http.Get("http://sock-server-1.cfapps.io/tunnel/" + d.DeviceId + "/api/capture"); err != nil {
		return nil, err
	} else {
		defer resp.Body.Close()
		return ioutil.ReadAll(resp.Body)
	}
}

func (d *Device) DetectFaces() (*DetectResponse, error) {
	if resp, err := http.Get("http://sock-server-1.cfapps.io/tunnel/" + d.DeviceId + "/api/detectFaces"); err != nil {
		return nil, err
	} else {
		defer resp.Body.Close()
		dr := DetectResponse{}
		if data, err := ioutil.ReadAll(resp.Body); err != nil {
			return nil, err
		} else {
			err := json.Unmarshal(data, &dr)
			return &dr, err
		}
	}
}

func (d *Device) RecognizeFaces() (*RecognizeResponse, error) {
	if resp, err := http.Get("http://sock-server-1.cfapps.io/tunnel/" + d.DeviceId + "/api/recognizeFaces"); err != nil {
		return nil, err
	} else {
		defer resp.Body.Close()
		rr := RecognizeResponse{}
		if data, err := ioutil.ReadAll(resp.Body); err != nil {
			return nil, err
		} else {
			err := json.Unmarshal(data, &rr)
			return &rr, err
		}
	}
}

func (d *Device) ReloadSamples() (error) {

	if resp, err := http.Post("http://192.168.40.137:9999/api/reloadSamples", "", nil); err != nil {
		return err
	} else {
		defer resp.Body.Close()
		if data, err := ioutil.ReadAll(resp.Body); err != nil {
			return err
		} else {
			if resp.StatusCode != 200 {
				return errors.New("fail to reload samples:" + string(data))
			}
			return nil
		}
	}

}
