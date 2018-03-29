package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
)

func mustOpen(f string) *os.File {
	r, err := os.Open(f)
	if err != nil {
		panic(err)
	}
	return r
}

func createMultipart(v map[string]io.Reader) (*bytes.Buffer, *multipart.Writer, error) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	var err error
	for key, r := range v {
		var fw io.Writer
		if x, ok := r.(io.Closer); ok {
			defer x.Close()
		}
		// Add an image file
		if x, ok := r.(*os.File); ok {
			if fw, err = w.CreateFormFile(key, x.Name()); err != nil {
				return nil, nil, err
			}
		} else {
			// Add other fields
			if fw, err = w.CreateFormField(key); err != nil {
				return nil, nil, err
			}
		}
		if _, err := io.Copy(fw, r); err != nil {
			return nil, nil, err
		}

	}

	// Don't forget to close the multipart writer.
	// If you don't close it, your request will be missing the terminating boundary.
	w.Close()
	return &b, w, nil
}

// {"success":true,"key":"UEwkag","link":"https://file.io/UEwkag","expiry":"14 days"}
// {"success":false,"error":400,"message":"Trouble uploading file"}
type fileIOResponse struct {
	Status  string `json:"status,omitempty"`
	Error   int    `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
	Key     string `json:"key,omitempty"`
	Link    string `json:"link,omitempty"`
	Expiry  string `json:"expiry,omitempty"`
}

// UploadFile handles uploading the specified file to File.IO and returns the file URL.
func UploadFile(c Content) (string, error) {

	b, mp, err := createMultipart(map[string]io.Reader{
		"file": mustOpen(c.Path),
	})
	// Now that you have a form, you can submit it to your handler.
	req, err := http.NewRequest("POST", "https://file.io?expires=1d", b)
	if err != nil {
		return "", fmt.Errorf("Error creating request: %v", err.Error())
	}
	// Don't forget to set the content type, this will contain the boundary.
	req.Header.Set("Content-Type", mp.FormDataContentType())

	client := &http.Client{}
	// Submit the request
	res, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Error creating request: %v", err.Error())
	}

	// Check the response
	if res.StatusCode != http.StatusOK {
		err = fmt.Errorf("bad status: %s", res.Status)
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("Unable to read File.io Response: %s", err.Error())
	}
	var resp fileIOResponse
	json.Unmarshal(body, &resp)
	if s, err := strconv.ParseBool(resp.Status); err == nil || s {
		return "", fmt.Errorf("Unable to upload file to File.io: %s", resp.Message)
	}
	return resp.Link, nil
}
