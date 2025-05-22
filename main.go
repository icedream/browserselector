//go:generate goversioninfo -icon=./resource/icon.ico ./resource/versioninfo.json

package main

import (
	"errors"
	"os"
	"os/exec"
	"regexp"
	"sort"

	"github.com/adrg/xdg"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
	"github.com/sqweek/dialog"
)

func sortConfigBrowserPriority(input []domain) (output []domain, err error) {
	sort.Slice(input[:], func(i, j int) bool {
		return input[i].Priority < input[j].Priority
	})
	output = input
	return
}

func getUrl(args []string, config configuration) (url string, err error) {
	if len(args) < 1 {
		err = errors.New("missing parameters")
		return
	}

	for index, element := range args {
		start, err := regexp.Compile("(http|https|ftp|ftps|ftpes|file).*")
		if err != nil {
			return "", err
		}
		if start.MatchString(element) {
			url = args[index]
			break
		}
	}

	if url == "" {
		err = errors.New("no url found")
		return
	}

	return
}

func getFqdnFromUrl(url string, config configuration) (protocol string, fqdn string, err error) {
	// Regex match FQDN
	// https?:\/\/([^\/]*)\/?.*
	r, err := regexp.Compile("(http|https|ftp|ftps|ftpes)://([^/]*)/?.*")
	if err != nil {
		return
	}
	matches := r.FindStringSubmatch(url)

	if len(matches) < 3 {
		err = errors.New("invalid url: " + url)
		return
	}

	protocol = matches[1]
	fqdn = matches[2]

	return
}

func main() {
	if err := run(); err != nil {
		dialog.Message("Failed to run: %s", err).Error()
		os.Exit(1)
	}
}

func run() (err error) {
	dir, err := homedir.Dir()
	if err != nil {
		return
	}

	// Load config
	var config configuration
	viper.AddConfigPath(".")
	viper.AddConfigPath(xdg.ConfigHome)
	for _, configDir := range xdg.ConfigDirs {
		viper.AddConfigPath(configDir)
	}
	viper.AddConfigPath(dir)
	viper.SetConfigName("browserselector")
	err = viper.ReadInConfig()
	if err != nil {
		return
	}
	err = viper.Unmarshal(&config)
	if err != nil {
		return
	}

	config.Domain, err = sortConfigBrowserPriority(config.Domain)
	if err != nil {
		return
	}

	// Get url from arguments
	url, err := getUrl(os.Args, config)
	if err != nil {
		return
	}

	_, fqdn, err := getFqdnFromUrl(url, config)
	if err != nil {
		return
	}

	// Check rules to select browser
	selector := len(config.Domain) - 1
	for index, element := range config.Domain {
		match, _ := regexp.MatchString(element.Regex, fqdn)
		if match {
			selector = index
			break
		}
	}

	// Check if browser exists
	if _, ok := config.Browser[config.Domain[selector].Browser]; !ok {
		return errors.New("no such browser defined: " + config.Domain[selector].Browser)
	}

	// Start browser
	var command = config.Browser[config.Domain[selector].Browser].Exec
	var cmdArgs []string
	if config.Browser[config.Domain[selector].Browser].Script == "" {
		// Exe + "FQDN"
		//cmdArgs = append(cmdArgs, "\""+url+"\"")
		cmdArgs = append(cmdArgs, url)
	} else {
		// Exe + Script + "FQDN"
		cmdArgs = append(cmdArgs, config.Browser[config.Domain[selector].Browser].Script, url)
	}

	cmd := exec.Command(command, cmdArgs...)
	err = cmd.Start()
	if err != nil {
		os.Exit(1)
	}
	err = cmd.Process.Release()
	if err != nil {
		os.Exit(1)
	}

	proc := cmd.Process

	err = proc.Release()

	return
}
