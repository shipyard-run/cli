package jumppad

import (
	"github.com/jumppad-labs/hclconfig/resources"
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/blueprint"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/build"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/cache"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/cert"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/container"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/copy"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/docs"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/exec"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/helm"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/http"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/ingress"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/k8s"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/network"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/nomad"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/null"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/random"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/template"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/terraform"
	sdk "github.com/jumppad-labs/plugin-sdk"
)

func init() {
	config.RegisterResource(blueprint.TypeBlueprint, &blueprint.Blueprint{}, &null.Provider{})
	config.RegisterResource(build.TypeBuild, &build.Build{}, &build.Provider{})
	config.RegisterResource(cache.TypeImageCache, &cache.ImageCache{}, &cache.Provider{})
	config.RegisterResource(cert.TypeCertificateCA, &cert.CertificateCA{}, &cert.CAProvider{})
	config.RegisterResource(cert.TypeCertificateLeaf, &cert.CertificateLeaf{}, &cert.LeafProvider{})
	config.RegisterResource(container.TypeContainer, &container.Container{}, &container.Provider{})
	config.RegisterResource(container.TypeSidecar, &container.Sidecar{}, &container.Provider{})
	config.RegisterResource(copy.TypeCopy, &copy.Copy{}, &copy.Provider{})
	config.RegisterResource(docs.TypeDocs, &docs.Docs{}, &docs.DocsProvider{})
	config.RegisterResource(docs.TypeChapter, &docs.Chapter{}, &null.Provider{})
	config.RegisterResource(docs.TypeTask, &docs.Task{}, &null.Provider{})
	config.RegisterResource(docs.TypeBook, &docs.Book{}, &null.Provider{})
	config.RegisterResource(exec.TypeExec, &exec.Exec{}, &exec.Provider{})
	config.RegisterResource(helm.TypeHelm, &helm.Helm{}, &helm.Provider{})
	config.RegisterResource(http.TypeHTTP, &http.HTTP{}, &http.Provider{})
	config.RegisterResource(ingress.TypeIngress, &ingress.Ingress{}, &ingress.Provider{})
	config.RegisterResource(k8s.TypeK8sCluster, &k8s.Cluster{}, &k8s.ClusterProvider{})
	config.RegisterResource(k8s.TypeK8sConfig, &k8s.Config{}, &k8s.ConfigProvider{})
	// add alias for k8s
	config.RegisterResource(k8s.TypeKubernetesCluster, &k8s.Cluster{}, &k8s.ClusterProvider{})
	config.RegisterResource(k8s.TypeKubernetesConfig, &k8s.Config{}, &k8s.ConfigProvider{})

	config.RegisterResource(network.TypeNetwork, &network.Network{}, &network.Provider{})
	config.RegisterResource(nomad.TypeNomadCluster, &nomad.NomadCluster{}, &nomad.ClusterProvider{})
	config.RegisterResource(nomad.TypeNomadJob, &nomad.NomadJob{}, &nomad.JobProvider{})
	config.RegisterResource(random.TypeRandomNumber, &random.RandomNumber{}, &random.RandomNumberProvider{})
	config.RegisterResource(random.TypeRandomID, &random.RandomID{}, &random.RandomIDProvider{})
	config.RegisterResource(random.TypeRandomUUID, &random.RandomUUID{}, &random.RandomUUIDProvider{})
	config.RegisterResource(random.TypeRandomPassword, &random.RandomPassword{}, &random.RandomPasswordProvider{})
	config.RegisterResource(random.TypeRandomCreature, &random.RandomCreature{}, &random.RandomCreatureProvider{})
	config.RegisterResource(cache.TypeRegistry, &cache.Registry{}, &null.Provider{})
	config.RegisterResource(template.TypeTemplate, &template.Template{}, &template.TemplateProvider{})
	config.RegisterResource(terraform.TypeTerraform, &terraform.Terraform{}, &terraform.TerraformProvider{})

	// register providers for the default types
	config.RegisterResource(resources.TypeModule, &resources.Module{}, &null.Provider{})
	config.RegisterResource(resources.TypeOutput, &resources.Output{}, &null.Provider{})
	config.RegisterResource(resources.TypeVariable, &resources.Variable{}, &null.Provider{})
	config.RegisterResource(resources.TypeLocal, &resources.Local{}, &null.Provider{})
}

// PluginRegisterResource is a function that registers a resource with the config package
// it is used by external plugins to register their resources
func PluginRegisterResource(name string, r types.Resource, p sdk.Provider) {
	config.RegisterResource(name, r, p)
}

// PluginLoadState is a function that enables external plugins to load the
// saved state of the configuration
func PluginLoadState() (sdk.Config, error) {
	return config.LoadState()
}
