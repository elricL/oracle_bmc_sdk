package oraclebmc_sdk

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/99designs/httpsignatures-go"
	"io/ioutil"
	"net/http"
	"time"
)

type oracle_config struct {
	user                         string
	tenancy                      string
	fingerprint                  string
	private_key                  string
	core_endpoint                string
	endpoint_blockstorage_api    string
	endpoint_identity_api        string
	endpoint_compute_api         string
	endpoint_virtual_network_api string
	endpoint_object_storage_api  string
	verify_ssl                   bool
	log_requests                 bool
	additional_user_agent        string
}

func NewConfig(user string, tenancy string, fingerprint string, signing_key string) *oracle_config {
	core_endpoint := "https://iaas.us-phoenix-1.oraclecloud.com/20160918"
	obj_endpoint := "https://objectstorage.us-phoenix-1.oraclecloud.com"
	endpoint_identity_api := "https://identity.us-phoenix-1.oraclecloud.com/20160918"
	return &oracle_config{
		user:                         user,
		tenancy:                      tenancy,
		fingerprint:                  fingerprint,
		private_key:                  signing_key,
		core_endpoint:                core_endpoint,
		endpoint_blockstorage_api:    core_endpoint,
		endpoint_compute_api:         core_endpoint,
		endpoint_virtual_network_api: core_endpoint,
		endpoint_object_storage_api:  obj_endpoint,
		endpoint_identity_api:        endpoint_identity_api}
}

func (config *oracle_config) getKey() string {
	return fmt.Sprintf("%s/%s/%s", config.tenancy, config.user, config.fingerprint)
}

type ComputeApi struct {
	Config *oracle_config
}

type Image struct {
	BaseImageId            string
	CompartmentId          string
	CreateImageAllowed     string
	DisplayName            string
	Id                     string
	LifeCycleState         string
	OperatingSystemVersion string
	TimeCreated            time.Time
}

type instancesList []Instance

type Instance struct {
	AvailabilityDomain string
	CompartmentId      string
	DisplayName        string
	Id                 string
	ImageId            string
	LifecycleState     string
	Metadata           map[string]string
	Region             string
	Shape              string
	TimeCreated        time.Time
}

func (computeApi *ComputeApi) GetInstance(instanceId string) (*Instance, error) {
	suffix := fmt.Sprintf("/instances/%s", instanceId)

	req, err := http.NewRequest("GET", computeApi.Config.core_endpoint+suffix, nil)
	if err != nil {
		return nil, err
	}
	inject_headers(computeApi.Config, req)

	client := &http.Client{}
	resp, err := client.Do(req)
	decoder := json.NewDecoder(resp.Body)
	var instance Instance
	err = decoder.Decode(&instance)
	return &instance, nil
}

func (computeApi *ComputeApi) ListImages(compartment_id string) (*[]*Image, error) {
	suffix := "/images"
	req, err := http.NewRequest("GET", computeApi.Config.core_endpoint+suffix, nil)
	if err != nil {
		return nil, err
	}
	url := req.URL
	q := url.Query()
	q.Set("compartmentId", compartment_id)
	url.RawQuery = q.Encode()

	inject_headers(computeApi.Config, req)

	client := &http.Client{}
	resp, err := client.Do(req)

	decoder := json.NewDecoder(resp.Body)
	var images []Instance
	err = decoder.Decode(&images)

	return images, nil
}

func inject_headers(oracleConfig *oracle_config, request *http.Request) {
	var required_headers []string
	request.Header.Set("host", request.URL.Host)
	if request.Method == "POST" || request.Method == "PUT" {

		required_headers = []string{
			httpsignatures.RequestTarget,
			"date",
			"host",
			"x-content-sha256",
			"content-type",
			"content-length"}

		body, _ := ioutil.ReadAll(request.Body)
		hash := sha256.New()
		hash.Write([]byte(body))
		content_body := hash.Sum(nil)

		request.Header.Set("x-content-sha256", string(content_body))
		request.Header.Set("content-length", string(len(body)))
		request.Header.Set("content-type", "application/json")
	} else {
		required_headers = []string{httpsignatures.RequestTarget, "date", "host"}
	}
	signer := httpsignatures.NewSigner(httpsignatures.AlgorithmRsaSha256, required_headers...)
	signer.AuthRequest(oracleConfig.getKey(), oracleConfig.private_key, request)
}