package config

import (
	"errors"
	"fmt"

	"github.com/claudiodangelis/qrcp/application"
	"github.com/claudiodangelis/qrcp/util"
	"github.com/manifoldco/promptui"
)

func chooseInterface(flags application.Flags) (string, error) {
	interfaces, err := util.Interfaces(flags.ListAllInterfaces)
	if err != nil {
		return "", err
	}
	if len(interfaces) == 0 {
		return "", errors.New("no interfaces found")
	}
	if len(interfaces) == 1 && !interactive {
		for name := range interfaces {
			fmt.Printf("only one interface found: %s, using this one\n", name)
			return name, nil
		}
	}
	// Map for pretty printing
	m := make(map[string]string)
	items := []string{}
	for name, ip := range interfaces {
		label := fmt.Sprintf("%s (%s)", name, ip)
		m[label] = name
		items = append(items, label)
	}
	// Add the "any" interface
	anyIP := "0.0.0.0"
	anyName := "any"
	anyLabel := fmt.Sprintf("%s (%s)", anyName, anyIP)
	m[anyLabel] = anyName
	items = append(items, anyLabel)
	prompt := promptui.Select{
		Items: items,
		Label: "Choose interface",
	}
	_, result, err := prompt.Run()
	if err != nil {
		return "", err
	}
	return m[result], nil
}
