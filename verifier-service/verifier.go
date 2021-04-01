package main

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"
)

const (
	MAPR_HOME                        = "/opt/mapr"
	MINIO_PID_FILE                   = MAPR_HOME + "/pid/objectstore.pid"
	OBJECTSTORE_HOME                 = MAPR_HOME + "/objectstore-client"
	OBJECTSTORE_VERSION_FILE_NAME    = "objectstoreversion"
	EXIT_CODE_NOT_RUNNING            = 1
	EXTIT_CODE_RUNNING_WITH_PROBLEMS = 2
)

func main() {
	versionFilePath := path.Join(OBJECTSTORE_HOME, OBJECTSTORE_VERSION_FILE_NAME)
	version, err := getVersion(versionFilePath)
	handleError(err, EXIT_CODE_NOT_RUNNING)

	objectstorePath := path.Join(OBJECTSTORE_HOME, "objectstore-client-"+version)
	setupLogger(objectstorePath)

	logger.Info("Objectstore version: ", version)

	objectstoreConfigDir := path.Join(objectstorePath, "conf")
	objectstoreConfigPath := path.Join(objectstoreConfigDir, "minio.json")

	config, err := getConfig(objectstoreConfigPath)
	handleError(err, EXIT_CODE_NOT_RUNNING)

	err = checkPid(MINIO_PID_FILE)
	handleError(err, EXIT_CODE_NOT_RUNNING)

	err = checkLogs(config.LogPath)
	handleError(err, EXTIT_CODE_RUNNING_WITH_PROBLEMS)

	err = checkPort(config.Port)
	handleError(err, EXTIT_CODE_RUNNING_WITH_PROBLEMS)

	url, err := getUrl(objectstoreConfigDir, config.Port)
	handleError(err, EXTIT_CODE_RUNNING_WITH_PROBLEMS)

	err = checkUI(url)
	handleError(err, EXTIT_CODE_RUNNING_WITH_PROBLEMS)

	mcPath := path.Join(objectstorePath, "util", "mc")
	mcAlias, err := configureMC(mcPath, url, config.AccessKey, config.SecretKey)
	handleError(err, EXTIT_CODE_RUNNING_WITH_PROBLEMS)

	err = checkServerMC(mcPath, mcAlias)
	handleError(err, EXTIT_CODE_RUNNING_WITH_PROBLEMS)

	err = checkLsMC(mcPath, mcAlias)
	handleError(err, EXTIT_CODE_RUNNING_WITH_PROBLEMS)

	err = removeAliasMC(mcPath, mcAlias)
	handleError(err, EXTIT_CODE_RUNNING_WITH_PROBLEMS)

	pathInHadoop, err := getMountPathInHadoop(config.FsPath)
	handleError(err, EXTIT_CODE_RUNNING_WITH_PROBLEMS)

	err = checkHadoopMinio(pathInHadoop)
	handleError(err, EXTIT_CODE_RUNNING_WITH_PROBLEMS)
}

func getVersion(path string) (version string, err error) {
	fmt.Println("Getting Objectstore version from ", path, " ...")

	version, err = readFileContent(path)

	return version, err
}

func checkPid(path string) (err error) {
	logger.Info("Checking pid file...")

	pidString, err := readFileContent(path)
	if err != nil {
		return err
	}

	pid, err := strconv.Atoi(pidString)
	if err != nil {
		return err
	}

	logger.Info("Checking pid ", pid, " ...")
	_, err = os.FindProcess(pid)

	if err != nil {
		return err
	}

	logger.Info("Pid OK")

	return err
}

type MapRMinioConfig struct {
	FsPath    string `json:"fsPath",omitempty`    /// Path to the Minio data root directory
	Port      string `json:"port",omitempty`      /// Port for server
	AccessKey string `json:"accessKey",omitempty` /// Minio accessKey
	SecretKey string `json:"secretKey",omitempty` /// Minio secretKey
	LogPath   string `json:"logPath",omitempty`   /// Path to the log file
}

func getConfig(path string) (config MapRMinioConfig, err error) {
	logger.Info("Reading config from ", path)

	data, err := readFileContent(path)
	if err != nil {
		return config, err
	}

	// Parsing config
	err = json.Unmarshal([]byte(data), &config)
	if err != nil {
		return config, errors.New(fmt.Sprintf("Failed to parse config: %s", err.Error()))
	}

	logger.Info("Config OK")

	return config, err
}

func checkLogs(path string) (err error) {
	logger.Info("Checking logs ", path, " ...")

	if _, err = os.Stat(path); os.IsNotExist(err) {
		return err
	}

	logger.Info("Logs OK")

	return nil
}

func checkPort(port string) (err error) {
	logger.Info("Checking port ", port, " ...")

	// Try to start listen port, if error, thaт port is already assigned
	if _, err = net.Listen("tcp", ":"+port); err != nil &&
		strings.Contains(err.Error(), "address already in use") {
		logger.Info("Port OK")
		return nil
	}

	return err
}

func getUrl(configDir, port string) (url string, err error) {
	logger.Info("Getting url ...")

	certsDir := path.Join(configDir, "certs")
	privateCertPath := path.Join(certsDir, "private.key")
	publicCertPath := path.Join(certsDir, "public.crt")
	privateCertExists := true
	publicCertExists := true

	logger.Info("Checking private certificate ", privateCertPath, " ...")
	if _, err = os.Stat(privateCertPath); os.IsNotExist(err) {
		privateCertExists = false
	}
	logger.Info("Private certificate exists:", privateCertExists)

	logger.Info("Checking public certificate ", privateCertPath, " ...")
	if _, err = os.Stat(publicCertPath); os.IsNotExist(err) {
		publicCertExists = false
	}
	logger.Info("Public certificate exists:", publicCertExists)

	// For using https we need both certificates
	if publicCertExists != privateCertExists {
		return url, errors.New("You should have both public and private certificates ")
	}

	protocol := ""
	if publicCertExists && privateCertExists {
		protocol = "https"
	} else {
		protocol = "http"
	}

	logger.Info("Protocol: ", protocol)

	url = fmt.Sprintf("%s://127.0.0.1:%s", protocol, port)

	return url, nil
}

func checkUI(url string) (err error) {
	logger.Info("Checking UI ", url, " ...")

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	// If UI is working correctly than we will receive error on login
	if resp.StatusCode != 403 {
		return errors.New(fmt.Sprintf("invalid status code: expected 403, but was %d", resp.StatusCode))
	}

	logger.Info("UI OK")

	return err
}

func configureMC(mcPath, url, accessKey, secretKey string) (aliasName string, err error) {
	logger.Info("Configuring MC ...")

	aliasName = "service_verifier"
	result, err := runCommand(mcPath, "alias", "set", aliasName, url, accessKey, secretKey, "--insecure", "--no-color")
	if err != nil {
		return aliasName, err
	}

	correctResult := fmt.Sprintf("Added `%s` successfully.", aliasName)
	if !strings.HasSuffix(result, correctResult) {
		return aliasName, errors.New(fmt.Sprintf("Failed to configure MC: %s", result))
	}

	logger.Info("MC configured successfully with alias `", aliasName, "`")

	return aliasName, nil
}

func checkServerMC(mcPath, aliasName string) (err error) {
	logger.Info("Checking servers info ...")

	stdout, err := runCommand(mcPath, "admin", "info", aliasName, "--insecure", "--no-color", "--json")
	if err != nil {
		return err
	}

	var result map[string]interface{}
	if err = json.Unmarshal([]byte(stdout), &result); err != nil {
		return err
	}

	status := result["status"]
	if status != "success" {
		return errors.New(fmt.Sprintf("Failed to get info via MC, status: %s", status))
	}
	logger.Info("Status OK")

	info := result["info"].(map[string]interface{})

	if info == nil {
		return errors.New(fmt.Sprintf("Failed to get info via MC, no info"))
	}

	mode := info["mode"]
	if mode != "online" {
		return errors.New(fmt.Sprintf("Failed to get info via MC, mode: %s", mode))
	}

	logger.Info("Mode OK")

	// Go through all servers list and checking their status
	servers := info["servers"].([]interface{})
	if servers == nil {
		return errors.New(fmt.Sprintf("Failed to get info via MC, no info about servers"))
	}
	for _, server := range servers {
		serverMap := server.(map[string]interface{})
		state := serverMap["state"]
		endpoint := serverMap["endpoint"]
		if state != "online" {
			return errors.New(fmt.Sprintf("Wrong state %s for server %s", state, endpoint))
		}
		logger.Info("Server ", endpoint, " OK")
	}

	return nil
}

func checkLsMC(mcPath, aliasName string) (err error) {
	logger.Info("Checking `ls` command ...")

	result, err := runCommand(mcPath, "ls", aliasName, "--insecure", "--no-color", "--json")
	if err != nil {
		return err
	}

	// Check if list of buckets is empty or containing objects
	if result == "" || (strings.HasPrefix(result, "{") && strings.HasSuffix(result, "}")) {
		logger.Info("`ls` OK")
		return nil
	}

	return errors.New(fmt.Sprintf("Failed to check `ls` MC: %s", result))
}

func removeAliasMC(mcPath, aliasName string) (err error) {
	logger.Info("Removing MC alias `", aliasName, "` ...")

	result, err := runCommand(mcPath, "alias", "remove", aliasName, "--insecure", "--no-color")
	if err != nil {
		return err
	}

	if result != "Removed `"+aliasName+"` successfully." {
		return errors.New(fmt.Sprintf("Failed to remove configurtion for MC: %s", result))
	}

	logger.Info("MC alias `", aliasName, "` removed successfully")

	return nil
}

func getMountPathInHadoop(mountPath string) (hadoopPath string, err error) {
	logger.Info("Searching for mount path in Hadoop for ", mountPath)

	result, err := runCommand("hadoop", "fs", "-ls", "/")

	if err != nil {
		return
	}

	resultArray := strings.Split(result, "\n")
	if len(resultArray) <= 1 {
		return hadoopPath, errors.New(fmt.Sprintf("Failed to found %s in hadoop fs: \n%s", mountPath, result))
	}

	hadoopPath = ""
	for i := 1; i < len(resultArray); i++ {
		line := resultArray[i]

		// Getting last part of output with folder name
		lastSpace := strings.LastIndex(line, " ")

		if lastSpace == -1 {
			continue
		}

		folderName := line[lastSpace+1:]

		// Cutting first found substring from path
		supPaths := strings.SplitN(mountPath, folderName, 2)
		if len(supPaths) == 2 {
			folder := path.Join(folderName, supPaths[1])

			// Finding the longest path, it have to be located in the root of MapR FS
			if len(folder) > len(hadoopPath) {
				hadoopPath = folder
			}
		}
	}

	if hadoopPath == "" {
		return hadoopPath, errors.New(fmt.Sprintf("Failed to found %s in hadoop fs: \n%s", mountPath, result))
	}

	logger.Info("Candidate for Hadoop mount is", hadoopPath)

	return hadoopPath, nil
}

func checkHadoopMinio(mountPath string) (err error) {
	logger.Info("Checking Objectstore directory in Hadoop ...")

	hadoopPath := path.Join(mountPath, ".minio.sys")
	logger.Info("Checking ", hadoopPath, " ...")

	result, err := runCommand("hadoop", "fs", "-ls", hadoopPath)
	if err != nil {
		return err
	}

	if strings.HasSuffix(result, "No such file or directory") {
		return errors.New(fmt.Sprintf("No %s in hadoop fs: %s", hadoopPath, result))
	}

	logger.Info("Hadoop OK")

	return nil
}

func readFileContent(path string) (content string, err error) {
	if _, err = os.Stat(path); os.IsNotExist(err) {
		return content, err
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return content, err
	}
	content = string(data)
	content = strings.TrimSuffix(content, "\n")

	return content, err
}

func runCommand(name string, arg ...string) (result string, err error) {
	stdout, err := exec.Command(name, arg...).Output()

	if err != nil {
		ee, ok := err.(*exec.ExitError)
		if ok {
			text := strings.TrimSuffix(string(ee.Stderr), "\n")
			text = fmt.Sprintf("%s: %s", name, text)
			return result, errors.New(text)
		}
		return result, err
	}

	result = strings.TrimSuffix(string(stdout), "\n")

	return result, nil
}

func handleError(err error, exitCode int) {
	if err != nil {
		logger.Error(err.Error())
		os.Exit(exitCode)
	}
}

type VerifierLogger struct {
	*log.Logger
}

var logger = New(os.Stderr, "", log.LstdFlags)

func setupLogger(objectstorePath string) {
	currentTime := time.Now()
	layout := "200601021504"
	timeStamp := currentTime.Format(layout)
	logFileName := fmt.Sprintf("verify_service.%s.log", timeStamp)
	logPath := path.Join(objectstorePath, "logs", logFileName)

	f, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		logger.Fatalf("error opening file: %v", err)
	}

	logger.SetOutput(f)
}

func New(out io.Writer, prefix string, flag int) *VerifierLogger {
	logOriginal := log.New(out, prefix, flag)
	return &VerifierLogger{Logger: logOriginal}
}

func (l *VerifierLogger) Info(v ...interface{}) {
	fmt.Println(v...)
	l.SetPrefix("INFO ")
	l.Output(2, fmt.Sprint(v...))
}

func (l *VerifierLogger) Error(v ...interface{}) {
	fmt.Println(v...)
	l.SetPrefix("ERROR ")
	l.Output(2, fmt.Sprint(v...))
}
