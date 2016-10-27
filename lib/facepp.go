package facepp

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strconv"
)

func getJson(responseBody io.Reader, target interface{}) error {
	return json.NewDecoder(responseBody).Decode(target)
}

type Facepp struct {
	server            string
	apiKey, apiSecret string
	useragent         string
}

func NewFacepp(server string, apiKey string, apiSecret string) Facepp {
	fpp := Facepp{useragent: "Idol Recognizer Go/0.1'"}
	fpp.server = server
	fpp.apiKey = apiKey
	fpp.apiSecret = apiSecret
	return fpp
}

func (fpp Facepp) Execute(method string, params map[string]string) (io.Reader, error) {
	v := url.Values{}

	for key, value := range params {
		v.Add(key, value)
	}

	v.Add("api_key", fpp.apiKey)
	v.Add("api_secret", fpp.apiSecret)

	resp, err := http.PostForm(fpp.server+method, v)

	if err != nil {
		return *new(io.Reader), err
	}
	if resp.StatusCode != 200 {
		return *new(io.Reader), errors.New(method + ": unexpected statuscode: " + strconv.Itoa(resp.StatusCode))
	}
	return resp.Body, err
}

func (fpp Facepp) ExecuteFileUpload(method string, params map[string]string, filepathS string) (io.Reader, error) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	f, err := os.Open(filepathS)
	if err != nil {
		return *new(io.Reader), err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return *new(io.Reader), err
	}
	if fi.Size() >= 3145728 {
		return *new(io.Reader), errors.New("image filesize bigger than 3 mb")
	}
	fw, err := w.CreateFormFile("img", filepathS)
	if err != nil {
		return *new(io.Reader), err
	}
	if _, err = io.Copy(fw, f); err != nil {
		return *new(io.Reader), err
	}

	params["api_key"] = fpp.apiKey
	params["api_secret"] = fpp.apiSecret

	for key, value := range params {
		if fw, err = w.CreateFormField(key); err != nil {
			return *new(io.Reader), err
		}
		if _, err = fw.Write([]byte(value)); err != nil {
			return *new(io.Reader), err
		}
	}

	w.Close()

	req, err := http.NewRequest("POST", fpp.server+method, &b)
	if err != nil {
		return *new(io.Reader), err
	}

	req.Header.Set("Content-Type", w.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return *new(io.Reader), err
	}

	if err != nil {
		return *new(io.Reader), err
	}
	if resp.StatusCode != 200 {
		return *new(io.Reader), errors.New(method + ": unexpected statuscode: " + strconv.Itoa(resp.StatusCode))
	}
	return resp.Body, err
}

type PersonList struct {
	Person []struct {
		Tag         string
		Person_name string
		Person_id   string
	}
}

func (fpp Facepp) InfoGetPersonList() (*PersonList, error) {
	target := new(PersonList)
	resp, err := fpp.Execute("/info/get_person_list", make(map[string]string))
	if err != nil {
		return target, err
	}
	err = json.NewDecoder(resp).Decode(target)
	return target, err
}

type PersonCreateO struct {
	Added_group int
	Added_face  int
	Tag         string
	Person_name string
	Person_id   string
}

func (fpp Facepp) PersonCreate(personName string, faceId string, tag string, groupName string) (*PersonCreateO, error) {
	target := new(PersonCreateO)
	params := make(map[string]string)
	if personName != "" {
		params["person_name"] = personName
	}
	if faceId != "" {
		params["face_id"] = faceId
	}
	if tag != "" {
		params["tag"] = tag
	}
	if groupName != "" {
		params["group_name"] = groupName
	}
	resp, err := fpp.Execute("/person/create", params)
	if err != nil {
		return target, err
	}
	err = json.NewDecoder(resp).Decode(target)
	return target, err
}

type DetectionDetectO struct {
	Session_id string
	Url        string
	Img_id     string
	Img_width  int
	Img_height int
	Face       []struct {
		Face_id string
		Tag     string
		/*Attribute []struct {
		      // TODO
		  }
		  Position []struct {
		      // TODO
		  }*/
	}
}

func (fpp Facepp) DetectionDetect(url string, mode string, tag string) (*DetectionDetectO, error) {
	target := new(DetectionDetectO)
	params := make(map[string]string)
	params["url"] = url
	if mode != "" {
		params["mode"] = mode
	}
	if tag != "" {
		params["tag"] = tag
	}
	resp, err := fpp.Execute("/detection/detect", params)
	if err != nil {
		return target, err
	}
	err = getJson(resp, target)
	return target, err
}

func (fpp Facepp) DetectionDetectFile(filepath string, mode string, tag string) (*DetectionDetectO, error) {
	target := new(DetectionDetectO)
	params := make(map[string]string)

	if mode != "" {
		params["mode"] = mode
	}
	if tag != "" {
		params["tag"] = tag
	} else {
		params["tag"] = filepath
	}
	resp, err := fpp.ExecuteFileUpload("/detection/detect", params, filepath)
	if err != nil {
		return target, err
	}
	err = getJson(resp, target)
	return target, err
}

type RecognitionIdentifyO struct {
	Session_id string
	Face       []struct {
		Face_id   string
		Candidate []struct {
			Confidence  float64
			Person_id   string
			Person_name string
			Tag         string
		}
		// position TODO
	}
}

func (fpp Facepp) RecognitionIdentify(groupName string, url string, mode string, keyFaceId string) (*RecognitionIdentifyO, error) {
	target := new(RecognitionIdentifyO)
	params := make(map[string]string)
	params["group_name"] = groupName
	if url != "" {
		params["url"] = url
	}
	if mode != "" {
		params["mode"] = mode
	}
	if keyFaceId != "" {
		params["key_face_id"] = keyFaceId
	}
	resp, err := fpp.Execute("/recognition/identify", params)
	if err != nil {
		return target, err
	}
	err = json.NewDecoder(resp).Decode(target)
	return target, err
}

func (fpp Facepp) RecognitionIdentifyFile(groupName string, filepath string, mode string, keyFaceId string) (*RecognitionIdentifyO, error) {
	target := new(RecognitionIdentifyO)
	params := make(map[string]string)

	params["group_name"] = groupName
	if mode != "" {
		params["mode"] = mode
	}
	if keyFaceId != "" {
		params["key_face_id"] = keyFaceId
	}
	resp, err := fpp.ExecuteFileUpload("/recognition/identify", params, filepath)
	if err != nil {
		return target, err
	}
	err = getJson(resp, target)
	return target, err
}

type GroupCreateO struct {
	Added_person int
	Group_id     string
	Group_name   string
	Tag          string
}

func (fpp Facepp) GroupCreate(groupName string, tag string, personName string) (*GroupCreateO, error) {
	target := new(GroupCreateO)
	params := make(map[string]string)
	params["group_name"] = groupName
	if tag != "" {
		params["tag"] = tag
	}
	if personName != "" {
		params["person_name"] = personName
	}
	resp, err := fpp.Execute("/group/create", params)
	if err != nil {
		return target, err
	}
	err = json.NewDecoder(resp).Decode(target)
	return target, err
}

type TrainIdentifyO struct {
	Session_id string
}

func (fpp Facepp) TrainIdentify(groupName string) (*TrainIdentifyO, error) {
	target := new(TrainIdentifyO)
	params := make(map[string]string)
	params["group_name"] = groupName
	resp, err := fpp.Execute("/train/identify", params)
	if err != nil {
		return target, err
	}
	err = json.NewDecoder(resp).Decode(target)
	return target, err
}

type InfoGetSessionO struct {
	Session_id  string
	Create_time int
	Finish_time int
	Status      string
	// Result TODO
}

func (fpp Facepp) InfoGetSession(sessionId string) (*InfoGetSessionO, error) {
	target := new(InfoGetSessionO)
	params := make(map[string]string)
	params["session_id"] = sessionId
	resp, err := fpp.Execute("/info/get_session", params)
	if err != nil {
		return target, err
	}
	err = json.NewDecoder(resp).Decode(target)
	return target, err
}

type PersonAddFaceO struct {
	Added   int
	Success bool
}

func (fpp Facepp) PersonAddFace(personName string, faceId string) (*PersonAddFaceO, error) {
	target := new(PersonAddFaceO)
	params := make(map[string]string)
	params["person_name"] = personName
	params["face_id"] = faceId
	resp, err := fpp.Execute("/person/add_face", params)
	if err != nil {
		return target, err
	}
	err = json.NewDecoder(resp).Decode(target)
	return target, err
}
