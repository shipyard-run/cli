package utils

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/jumppad-labs/jumppad/pkg/utils/dirhash"
	"github.com/kennygrant/sanitize"
)

// EnsureAbsolute ensure that the given path is either absolute or
// if relative is converted to abasolute based on the path of the config
func EnsureAbsolute(path, file string) string {
	// do not try to convert the docker sock address
	// this could happen if someone mounts the docker sock on windows
	if path == GetDockerHost() {
		return path
	}

	// if the file starts with a / and we are on windows
	// we should treat this as absolute
	if runtime.GOOS == "windows" && strings.HasPrefix(path, "/") {
		return filepath.Clean(path)
	}

	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}

	// path is relative so make absolute using the current file path as base
	file, _ = filepath.Abs(file)

	baseDir := file
	// check if the basepath is a file return its directory
	s, _ := os.Stat(file)
	if !s.IsDir() {
		baseDir = filepath.Dir(file)
	}

	fp := filepath.Join(baseDir, path)

	return filepath.Clean(fp)
}

// Creates the required file structure in the users Home directory
func CreateFolders() {
	os.MkdirAll(ReleasesFolder(), os.FileMode(0755))
}

// ValidateName ensures that the name for a resource is within certain boundaries
// Valid characters: [a-z] [A-Z] _ - [0-9]
// Max length: 128
func ValidateName(name string) (bool, error) {
	// check the length
	if len(name) > 128 {
		return false, NameExceedsMaxLengthError
	}

	r := regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`)
	ok := r.MatchString(name)
	if !ok {
		return false, NameContainsInvalidCharactersError
	}

	return true, nil
}

// ReplaceNonURIChars replaces any characters in the resource name which
// can not be used in a URI
func ReplaceNonURIChars(s string) (string, error) {
	reg, err := regexp.Compile(`[^a-zA-Z0-9\-\.]+`)
	if err != nil {
		return "", err
	}

	ret := reg.ReplaceAllString(s, "-")

	if strings.HasPrefix(ret, "-") {
		return ret[1:], nil
	}

	return ret, nil
}

// FQDN generates the full qualified name for a container
func FQDN(name, module, typeName string) string {
	fqdn := fmt.Sprintf("%s.%s.local.%s", name, typeName, LocalTLD)
	if module != "" {
		fqdn = fmt.Sprintf("%s.%s.%s.local.%s", name, module, typeName, LocalTLD)
	}

	// ensure that the name is valid for URI schema
	cleanName, err := ReplaceNonURIChars(fqdn)
	if err != nil {
		panic(err)
	}

	return cleanName
}

// FQDNVolumeName creates a full qualified volume name
func FQDNVolumeName(name string) string {
	// ensure that the name is valid for URI schema
	cleanName, err := ReplaceNonURIChars(name)
	if err != nil {
		panic(err)
	}

	return fmt.Sprintf("%s.volume.%s", cleanName, LocalTLD)
}

// CreateKubeConfigPath creates the file path for the KubeConfig file when
// using Kubernetes cluster
func CreateKubeConfigPath(id string) (dir, filePath string, dockerPath string) {
	id, _ = ReplaceNonURIChars(id)
	dir = filepath.Join(JumppadHome(), "/config/", id)
	filePath = filepath.Join(dir, "/kubeconfig.yaml")
	dockerPath = filepath.Join(dir, "/kubeconfig-docker.yaml")

	// create the folders
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		panic(err)
	}

	return
}

// HomeFolder returns the users homefolder this will be $HOME on windows and mac and
// USERPROFILE on windows
func HomeFolder() string {
	return os.Getenv(HomeEnvName())
}

// HomeEnvName returns the environment variable used to store the home path
func HomeEnvName() string {
	if runtime.GOOS == "windows" {
		return "USERPROFILE"
	}

	return "HOME"
}

// JumppadHome returns the location of the jumppad
// folder, usually $HOME/.jumppad
func JumppadHome() string {
	return filepath.Join(HomeFolder(), "/.jumppad")
}

// JumppadTemp returns a temporary folder
func JumppadTemp() string {
	dir := filepath.Join(JumppadHome(), "/tmp")
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		panic(err)
	}

	return dir
}

// StateDir returns the location of the jumppad
// state, usually $HOME/.jumppad/state
func StateDir() string {
	return filepath.Join(JumppadHome(), "/state")
}

// PluginsDir returns the location of the plugins
func PluginsDir() string {
	logs := filepath.Join(JumppadHome(), "/plugins")

	os.MkdirAll(logs, os.ModePerm)
	return logs
}

// CertsDir returns the location of the certificates for the given resource
// used to secure the Jumppad ingress, usually rooted at $HOME/.jumppad/certs
func CertsDir(name string) string {
	certs := filepath.Join(JumppadHome(), "/certs", name)
	certs = filepath.FromSlash(certs)

	// create the folder if it does not exist
	os.MkdirAll(certs, os.ModePerm)
	return certs
}

// LogsDir returns the location of the logs
// used to secure the Jumppad ingress, usually $HOME/.jumppad/logs
func LogsDir() string {
	logs := filepath.Join(JumppadHome(), "/logs")

	os.MkdirAll(logs, os.ModePerm)
	return logs
}

// StatePath returns the full path for the state file
func StatePath() string {
	return filepath.Join(StateDir(), "/state.json")
}

// ImageCacheLog returns the location of the image cache log
func ImageCacheLog() string {
	return fmt.Sprintf("%s/images.log", JumppadHome())
}

// IsLocalFolder tests if the given path is a localfolder and can
// exist in the current filesystem
// TODO make more robust with error messages
// to improve UX
func IsLocalFolder(path string) bool {
	path, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	f, err := os.Stat(path)
	if err != nil || f == nil {
		return false
	}

	return true
}

// IsHCLFile tests if the given path resolves to a HCL config file
func IsHCLFile(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}

	if s.IsDir() {
		return false
	}

	if filepath.Ext(s.Name()) != ".hcl" {
		return false
	}

	return true
}

// BlueprintFolder parses a blueprint uri and returns the top level
// blueprint folder
// if the URI is not a blueprint will return an error
func BlueprintFolder(blueprint string) (string, error) {
	// get the folder for the blueprint
	parts := strings.Split(blueprint, "//")

	if parts == nil || len(parts) != 2 {
		return "", InvalidBlueprintURIError
	}

	// first replace any ?
	parts[1] = strings.Replace(parts[1], "?", "-", -1)

	return sanitize.Path(parts[1]), nil
}

// BlueprintLocalFolder returns the full storage path
// for the given blueprint URI
func BlueprintLocalFolder(blueprint string) string {
	// we might have a querystring reference such has github.com/abc/cds?ref=dfdf&dfdf
	// replace these separators with /

	// replace any ? with / before sanitizing
	blueprint = strings.Replace(blueprint, "?", "/", -1)

	blueprint = sanitize.Path(blueprint)

	return filepath.Join(JumppadHome(), "blueprints", blueprint)
}

// HelmLocalFolder returns the full storage path
// for the given blueprint URI
func HelmLocalFolder(chart string) string {
	// replace any ? with / before sanitizing
	chart = strings.Replace(chart, "?", "/", -1)

	chart = sanitize.Path(chart)

	return filepath.Join(JumppadHome(), "helm_charts", chart)
}

// ReleasesFolder return the path of the Shipyard releases
func ReleasesFolder() string {
	return filepath.Join(JumppadHome(), "releases")
}

// DataFolder creates the data directory used by the application
func DataFolder(p string, perms os.FileMode) string {
	data := filepath.Join(JumppadHome(), "data", p)

	// create the folder if it does not exist
	os.MkdirAll(data, perms)
	os.Chmod(data, perms)

	return data
}

// CacheFolder creates the cache directory used by the a provider
// unlike DataFolders, cache folders are not removed when down is called
func CacheFolder(p string, perms os.FileMode) string {
	data := filepath.Join(JumppadHome(), "cache", p)

	// create the folder if it does not exist
	os.MkdirAll(data, perms)
	os.Chmod(data, perms)

	return data
}

// LibraryFolder creates the library directory used by the application
func LibraryFolder(p string, perms os.FileMode) string {
	p = sanitize.Path(p)
	data := filepath.Join(JumppadHome(), "library", p)

	// create the folder if it does not exist
	os.MkdirAll(data, perms)
	os.Chmod(data, perms)

	return data
}

// GetDockerHost returns the location of the Docker API depending on the platform
func GetDockerHost() string {
	if dh := os.Getenv("DOCKER_HOST"); dh != "" {
		return dh
	}

	return "unix:///var/run/docker.sock"
}

// GetDockerIP returns the location of the Docker Server IP address
func GetDockerIP() string {
	if dh := os.Getenv("DOCKER_HOST"); dh != "" {
		if strings.HasPrefix(dh, "tcp://") {
			u, err := url.Parse(dh)
			if err == nil {
				host := strings.Split(u.Host, ":")[0]
				ip, err := net.LookupHost(host)
				if err == nil && len(ip) > 0 {
					return ip[0]
				}
			}
		}
	}

	sp, _ := GetLocalIPAndHostname()

	return sp
}

// GetConnectorPIDFile returns the connector PID file used by the connector
func GetConnectorPIDFile() string {
	return filepath.Join(JumppadHome(), "connector.pid")
}

// GetConnectorLogFile returns the log file used by the connector
func GetConnectorLogFile() string {
	return filepath.Join(LogsDir(), "connector.log")
}

// GetJumppadBinaryPath returns the path to the running Jumppad binary
func GetJumppadBinaryPath() string {
	exe, _ := os.Executable()

	return exe
}

// GetHostname returns the hostname for the current machine
func GetHostname() string {
	hn, err := os.Hostname()
	if err != nil {
		return ""
	}

	return hn
}

// GetLocalIPAddress returns a list of ip addressses for the local machine
func GetLocalIPAddresses() []string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return []string{}
	}

	addresses := []string{}
	for _, a := range addrs {
		ip, _, err := net.ParseCIDR(a.String())
		if err == nil {
			addresses = append(addresses, fmt.Sprintf("%s", ip))
		}
	}

	return addresses
}

// GetLocalIPAndHostname returns the IP Address of the machine
func GetLocalIPAndHostname() (string, string) {
	netInterfaceAddresses, err := net.InterfaceAddrs()
	if err != nil {
		return "", ""
	}

	for _, netInterfaceAddress := range netInterfaceAddresses {
		networkIP, ok := netInterfaceAddress.(*net.IPNet)
		if ok && !networkIP.IP.IsLoopback() && networkIP.IP.To4() != nil {
			ip := networkIP.IP.String()
			return ip, GetHostname()
		}
	}

	return "127.0.0.1", "localhost"
}

// ImageCacheADDR returns the default Image cache used by
// Nomad and Kubernetes clusters unless the environment variable
// IMAGE_CACHE_ADDR is set when it returns this value
func ImageCacheAddress() string {
	if p := os.Getenv("IMAGE_CACHE_ADDR"); p != "" {
		return p
	}

	return jumppadProxyAddress
}

// get all ipaddresses in a subnet
func SubnetIPs(subnet string) ([]string, error) {
	_, ipnet, _ := net.ParseCIDR(subnet)

	var ipList []string
	ip := ipnet.IP
	for ; ipnet.Contains(ip); ip = incIP(ip) {
		ipList = append(ipList, ip.String())
	}

	return ipList, nil
}

// HashDir generates a hash of the given directory
// optionally a list of arguments to be ignored can be passed
// these arguments are expresed as a glob pattern
func HashDir(dir string, ignore ...string) (string, error) {
	return dirhash.HashDir(dir, "", dirhash.DefaultHash, ignore...)
}

// HashFile returns a sha256 hash of the given file
func HashFile(file string) (string, error) {
	r, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer r.Close()

	hf := sha256.New()
	_, err = io.Copy(hf, r)

	if err != nil {
		return "", err
	}

	return "h1:" + base64.StdEncoding.EncodeToString(hf.Sum(nil)), nil
}

// HashString returns a sha256 hash of the given string
func HashString(content string) (string, error) {
	r := bytes.NewReader([]byte(content))

	hf := sha256.New()
	_, err := io.Copy(hf, r)
	if err != nil {
		return "", err
	}

	return "h1:" + base64.StdEncoding.EncodeToString(hf.Sum(nil)), nil
}

// InterfaceChecksum returns a checksum of the given interface
// Note: the checksum is positional, should an element in a map or list change
// position then a different checksum will be returned.
func ChecksumFromInterface(i interface{}) (string, error) {
	// first convert the object to json
	json, err := json.Marshal(i)
	if err != nil {
		return "", fmt.Errorf("unable to marshal interface: %w", err)
	}

	return HashString(string(json))
}

// RandomAvailablePort returns a random free port in the given range
func RandomAvailablePort(from, to int) (int, error) {

	// checks 10 times for a free port
	for i := 0; i < 10; i++ {
		port := rand.Intn(to-from) + from

		// check if the port is available
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			ln.Close()

			return port, nil
		}
	}

	return 0, fmt.Errorf("unable to find a free port in the range %d-%d", from, to)
}

func incIP(ip net.IP) net.IP {
	// allocate a new IP
	newIp := make(net.IP, len(ip))
	copy(newIp, ip)

	byteIp := []byte(newIp)
	l := len(byteIp)
	var i int
	for k, _ := range byteIp {
		// start with the rightmost index first
		// increment it
		// if the index is < 256, then no overflow happened and we increment and break
		// else, continue to the next field in the byte
		i = l - 1 - k
		if byteIp[i] < 0xff {
			byteIp[i]++
			break
		} else {
			byteIp[i] = 0
		}
	}
	return net.IP(byteIp)
}
