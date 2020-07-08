package task

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/docker/docker/api/types"
)

type configFile struct {
	AuthConfigs          map[string]types.AuthConfig `json:"auths"`
	HTTPHeaders          map[string]string           `json:"HttpHeaders,omitempty"`
	PsFormat             string                      `json:"psFormat,omitempty"`
	ImagesFormat         string                      `json:"imagesFormat,omitempty"`
	NetworksFormat       string                      `json:"networksFormat,omitempty"`
	PluginsFormat        string                      `json:"pluginsFormat,omitempty"`
	VolumesFormat        string                      `json:"volumesFormat,omitempty"`
	StatsFormat          string                      `json:"statsFormat,omitempty"`
	DetachKeys           string                      `json:"detachKeys,omitempty"`
	CredentialsStore     string                      `json:"credsStore,omitempty"`
	CredentialHelpers    map[string]string           `json:"credHelpers,omitempty"`
	Filename             string                      `json:"-"` // Note: for internal use only
	ServiceInspectFormat string                      `json:"serviceInspectFormat,omitempty"`
	ServicesFormat       string                      `json:"servicesFormat,omitempty"`
	TasksFormat          string                      `json:"tasksFormat,omitempty"`
	SecretFormat         string                      `json:"secretFormat,omitempty"`
	ConfigFormat         string                      `json:"configFormat,omitempty"`
	NodesFormat          string                      `json:"nodesFormat,omitempty"`
	PruneFilters         []string                    `json:"pruneFilters,omitempty"`
}

func exitErrorf(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg+"\n", args...)
	os.Exit(1)
}

// Get list of registries from env, leave empty for default
func getRegistryIds() []string {
	var registryIds []string
	registries, exists := os.LookupEnv("REGISTRIES")
	if exists {
		for _, registry := range strings.Split(registries, ",") {
			registryIds = append(registryIds, *aws.String(registry))
		}
	}
	return registryIds
}

// GetECRLogin Creates Docker config.json via IAM role
func GetECRLogin() {
	var region = "eu-west-1"

	// Create the ECR service config
	awscfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		exitErrorf("failed to load config, %v", err)
	}
	if len(region) > 0 {
		awscfg.Region = region
	}

	// Handle multiple registries
	params := &ecr.GetAuthorizationTokenInput{
		RegistryIds: getRegistryIds(),
	}

	// Create the ECR service client
	svc := ecr.New(awscfg)
	req := svc.GetAuthorizationTokenRequest(params)
	resp, err := req.Send(context.TODO())
	if err != nil {
		fmt.Println(err)
	} else {
		var authConfig types.AuthConfig
		config := configFile{}
		config.AuthConfigs = make(map[string]types.AuthConfig)
		for _, auth := range resp.AuthorizationData {
			authConfig.Auth = *auth.AuthorizationToken
			authConfig.Email = "concourse@iptho.co.uk"
			config.AuthConfigs[*auth.ProxyEndpoint] = authConfig
		}

		configJSON, err := json.Marshal(config)
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println(string(configJSON))

		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Println(err)
		} else {
			configDir := filepath.Join(homeDir, ".docker")
			err := os.Setenv("DOCKER_CONFIG", configDir)
			if _, err := os.Stat(configDir); os.IsNotExist(err) {
				os.Mkdir(configDir, 0744)
			}
			if err != nil {
				fmt.Println(err)
			} else {
				configFile := filepath.Join(configDir, "config.json")
				err := ioutil.WriteFile(configFile, configJSON, 0644)
				if err != nil {
					fmt.Println(err)
				}
			}
		}
	}
}
