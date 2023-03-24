package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/hclconfig"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/config/resources"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"github.com/spf13/cobra"
)

var resumeCmd = &cobra.Command{
	Use:   "resume",
	Short: "Resume a paused session and restart all resources",
	Long:  `Resume a paused session and restart all resources`,
	Example: `
  shipyard resume
	`,
	Args: cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		fmt.Println("Resuming session")

		l := createLogger()

		// create a docker client
		c, err := clients.NewDocker()
		if err != nil {
			l.Error("Unable to connect to Docker daemon", "error", err)
			os.Exit(1)
		}

		cl, err := getContainers(c, "exited")
		if err != nil {
			l.Error("Unable to get container status", "error", err)
			os.Exit(1)
		}

		// start the containers
		for _, con := range cl {
			err := c.ContainerStart(context.Background(), con.ID, types.ContainerStartOptions{})
			if err != nil {
				l.Error("Unable to start container", "name", con.Names[0], "error", err)
				os.Exit(1)
			}
		}

		l.Info("Checking health of containers")
		// wait for containers to get healthy
		_, err = checkStatus(c)
		if err != nil {
			l.Error("Uable to check health of containers", "error", err)
			os.Exit(1)
		}

		// get the health checks from the config and test
		p := hclconfig.NewParser(hclconfig.DefaultOptions())
		d, err := ioutil.ReadFile(utils.StatePath())
		if err != nil {
			l.Error("Unable to read state file")
			os.Exit(1)
		}

		cfg, err := p.UnmarshalJSON(d)
		if err != nil {
			l.Error("Unable to unmarshal state file")
		}

		for _, res := range cfg.Resources {
			switch res.Metadata().Type {
			case resources.TypeHelm:
				co := res.(*resources.Helm)
				hc := co.HealthCheck

				if hc != nil && len(hc.Pods) != 0 {
					l.Debug("Health check pods in Helm chart", "chart", co.Metadata().Name)
					err := healthCheckHelm(co)
					if err != nil {
						l.Error("Unable to check health of helm chart", "error", err)
						os.Exit(1)
					}
				}
			case resources.TypeK8sConfig:
				co := res.(*resources.K8sConfig)
				hc := co.HealthCheck

				if hc != nil && len(hc.Pods) != 0 {
					l.Debug("Health check pods in Kubernetes config", "chart", co.Metadata().Name)
					err := healthCheckK8sConfig(co)
					if err != nil {
						l.Error("Unable to check health of k8s_config chart", "error", err)
						os.Exit(1)
					}
				}
			}

		}

	},
}

func checkStatus(c clients.Docker) (bool, error) {
	st := time.Now()

	for {
		if time.Now().Sub(st) > (60 * time.Second) {
			return false, fmt.Errorf("Health check timeout waiting for containers to start failed")
		}

		// get the container status and check if running
		cl, err := getContainers(c, "")
		if err != nil {
			return false, err
		}

		allRunning := true
		for _, con := range cl {
			if con.State != "running" {
				allRunning = false
				break
			}
		}

		if allRunning {
			return true, nil
		}

		// wait 1s then try again
		time.Sleep(1 * time.Second)
	}
}

func getContainers(c clients.Docker, status string) ([]types.Container, error) {
	filters := filters.NewArgs()
	filters.Add("name", "shipyard")

	if status != "" {
		filters.Add("status", status)
	}

	cl, err := c.ContainerList(
		context.Background(),
		types.ContainerListOptions{
			Filters: filters,
		},
	)

	if err != nil {
		return nil, err
	}

	return cl, nil
}

// TODO: HealthChecks should really be moved to a central universal functional call
// copy pasta for now
func healthCheckHelm(h *resources.Helm) error {
	kc := clients.NewKubernetes(500*time.Second, hclog.Default())
	cl, err := h.Metadata().ParentConfig.FindResource(h.Cluster)
	if err != nil {
		return nil
	}

	_, conf, _ := utils.CreateKubeConfigPath(cl.Metadata().Name)
	kc, err = kc.SetConfig(conf)
	if err != nil {
		return nil
	}

	err = kc.HealthCheckPods(h.HealthCheck.Pods, 500*time.Second)
	if err != nil {
		return err
	}

	return nil
}

func healthCheckK8sConfig(h *resources.K8sConfig) error {
	kc := clients.NewKubernetes(500*time.Second, hclog.Default())
	cl, err := h.Metadata().ParentConfig.FindResource(h.Cluster)
	if err != nil {
		return nil
	}

	_, conf, _ := utils.CreateKubeConfigPath(cl.Metadata().Name)
	kc, err = kc.SetConfig(conf)
	if err != nil {
		return nil
	}

	err = kc.HealthCheckPods(h.HealthCheck.Pods, 500*time.Second)
	if err != nil {
		return err
	}

	return nil
}
