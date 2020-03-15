package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	metalcloud "github.com/bigstepinc/metal-cloud-sdk-go"
	interfaces "github.com/bigstepinc/metalcloud-cli/interfaces"
)

//instanceCmds commands affecting instances
var instanceCmds = []Command{

	Command{
		Description:  "Control power an instance",
		Subject:      "instance",
		AltSubject:   "instance",
		Predicate:    "power_control",
		AltPredicate: "pwr",
		FlagSet:      flag.NewFlagSet("instance_array", flag.ExitOnError),
		InitFunc: func(c *Command) {
			c.Arguments = map[string]interface{}{
				"instance_id": c.FlagSet.Int("id", _nilDefaultInt, "(Required) Instances's id . Note that the 'label' this be ambiguous in certain situations."),
				"operation":   c.FlagSet.String("operation", _nilDefaultStr, "(Required) Power control operation, one of: on, off, reset, soft"),
				"autoconfirm": c.FlagSet.Bool("autoconfirm", false, "If true it does not ask for confirmation anymore"),
			}
		},
		ExecuteFunc: instancePowerControlCmd,
	},

	Command{
		Description:  "Show an instance's credentials",
		Subject:      "instance",
		AltSubject:   "instance",
		Predicate:    "credentials",
		AltPredicate: "creds",
		FlagSet:      flag.NewFlagSet("instance credentials", flag.ExitOnError),
		InitFunc: func(c *Command) {
			c.Arguments = map[string]interface{}{
				"instance_id": c.FlagSet.Int("id", _nilDefaultInt, "(Required) Instances's id . Note that the 'label' this be ambiguous in certain situations."),
				"format":      c.FlagSet.String("format", "", "The output format. Supported values are 'json','csv'. The default format is human readable."),
			}
		},
		ExecuteFunc: instanceCredentialsCmd,
	},
}

func instancePowerControlCmd(c *Command, client interfaces.MetalCloudClient) (string, error) {

	instanceID, ok := getIntParamOk(c.Arguments["instance_id"])
	if !ok {
		return "", fmt.Errorf("-id is required (drive id)")
	}
	operation, ok := getStringParamOk(c.Arguments["operation"])
	if !ok {
		return "", fmt.Errorf("-operation is required (one of: on, off, reset, soft)")
	}

	instance, err := client.InstanceGet(instanceID)
	if err != nil {
		return "", err
	}

	ia, err := client.InstanceArrayGet(instance.InstanceArrayID)
	if err != nil {
		return "", err
	}

	infra, err := client.InfrastructureGet(ia.InfrastructureID)
	if err != nil {
		return "", err
	}

	confirm, err := confirmCommand(c, func() string {

		op := ""
		switch operation {
		case "on":
			op = "Turning on"
		case "off":
			op = "Turning off (hard)"
		case "reset":
			op = "Rebooting"
		case "sort":
			op = "Shutting down"
		}

		confirmationMessage := fmt.Sprintf("%s instance %s (%d) of instance array %s (#%d) infrastructure %s (#%d).  Are you sure? Type \"yes\" to continue:",
			op,
			instance.InstanceLabel,
			instance.InstanceID,
			ia.InstanceArrayLabel,
			ia.InstanceArrayID,
			infra.InfrastructureLabel,
			infra.InfrastructureID,
		)

		//this is simply so that we don't output a text on the command line under go test
		if strings.HasSuffix(os.Args[0], ".test") {
			confirmationMessage = ""
		}

		return confirmationMessage

	})

	if err != nil {
		return "", err
	}

	if confirm {
		err = client.InstanceServerPowerSet(instanceID, operation)
	}

	return "", err
}

func instanceCredentialsCmd(c *Command, client interfaces.MetalCloudClient) (string, error) {

	instanceID, ok := getIntParamOk(c.Arguments["instance_id"])
	if !ok {
		return "", fmt.Errorf("-id is required (instance id)")
	}

	instance, err := client.InstanceGet(instanceID)
	if err != nil {
		return "", err
	}

	ia, err := client.InstanceArrayGet(instance.InstanceArrayID)
	if err != nil {
		return "", err
	}

	infra, err := client.InfrastructureGet(ia.InfrastructureID)
	if err != nil {
		return "", err
	}

	schema := []SchemaField{

		SchemaField{
			FieldName: "ID",
			FieldType: TypeInt,
			FieldSize: 6,
		},
		SchemaField{
			FieldName: "SUBDOMAIN",
			FieldType: TypeString,
			FieldSize: 10,
		},
		SchemaField{
			FieldName: "INSTANCE_ARRAY",
			FieldType: TypeString,
			FieldSize: 10,
		},
		SchemaField{
			FieldName: "INFRASTRUCTURE",
			FieldType: TypeString,
			FieldSize: 10,
		},
		SchemaField{
			FieldName: "PUBLIC_IPs",
			FieldType: TypeString,
			FieldSize: 6,
		},
		SchemaField{
			FieldName: "PRIVATE_IPs",
			FieldType: TypeString,
			FieldSize: 6,
		},
	}

	publicIPS := getIPsAsStringArray(instance.InstanceCredentials.IPAddressesPublic)
	privateIPS := getIPsAsStringArray(instance.InstanceCredentials.IPAddressesPrivate)

	dataRow := []interface{}{
		instance.InstanceID,
		instance.InstanceSubdomainPermanent,
		ia.InstanceArrayLabel,
		infra.InfrastructureLabel,
		strings.Join(publicIPS, " "),
		strings.Join(privateIPS, " "),
	}

	if v := instance.InstanceCredentials.SSH; v != nil {

		newFields := []SchemaField{
			SchemaField{
				FieldName: "SSH_USERNAME",
				FieldType: TypeString,
				FieldSize: 10,
			},
			SchemaField{
				FieldName: "SSH_PASSWORD",
				FieldType: TypeString,
				FieldSize: 10,
			},
			SchemaField{
				FieldName: "SSH_PORT",
				FieldType: TypeInt,
				FieldSize: 10,
			},
		}

		schema = append(schema, newFields...)

		newData := []interface{}{
			v.Username,
			v.InitialPassword,
			v.Port,
		}
		dataRow = append(dataRow, newData...)
	}

	if v := instance.InstanceCredentials.RDP; v != nil {

		newFields := []SchemaField{
			SchemaField{
				FieldName: "RDP_USERNAME",
				FieldType: TypeString,
				FieldSize: 5,
			},
			SchemaField{
				FieldName: "RDP_PASSWORD",
				FieldType: TypeString,
				FieldSize: 5,
			},
			SchemaField{
				FieldName: "RDP_PORT",
				FieldType: TypeInt,
				FieldSize: 5,
			},
		}

		schema = append(schema, newFields...)
		newData := []interface{}{
			v.Username,
			v.InitialPassword,
			v.Port,
		}
		dataRow = append(dataRow, newData...)
	}

	if v := instance.InstanceCredentials.ISCSI; v != nil {

		newFields := []SchemaField{
			SchemaField{
				FieldName: "INITIATOR_IQN",
				FieldType: TypeString,
				FieldSize: 5,
			},
			SchemaField{
				FieldName: "ISCSI_USERNAME",
				FieldType: TypeString,
				FieldSize: 5,
			},
			SchemaField{
				FieldName: "ISCSI_PASSWORD",
				FieldType: TypeString,
				FieldSize: 5,
			},
		}

		schema = append(schema, newFields...)
		newData := []interface{}{
			v.InitiatorIQN,
			v.Username,
			v.Password,
		}
		dataRow = append(dataRow, newData...)
	}

	if v := instance.InstanceCredentials.SharedDrives; v != nil {

		for k, sd := range v {
			newFields := []SchemaField{
				SchemaField{
					FieldName: fmt.Sprintf("SHARED_DRIVE_%d_TARGET_IP_ADDRESS", k),
					FieldType: TypeString,
					FieldSize: 5,
				},
				SchemaField{
					FieldName: fmt.Sprintf("SHARED_DRIVE_%d_TARGET_PORT", k),
					FieldType: TypeInt,
					FieldSize: 5,
				},
				SchemaField{
					FieldName: fmt.Sprintf("SHARED_DRIVE_%d_TARGET_IQN", k),
					FieldType: TypeString,
					FieldSize: 5,
				},
				SchemaField{
					FieldName: fmt.Sprintf("SHARED_DRIVE_%d_LUN_ID", k),
					FieldType: TypeString,
					FieldSize: 5,
				},
			}

			schema = append(schema, newFields...)
			newData := []interface{}{
				sd.StorageIPAddress,
				sd.StoragePort,
				sd.TargetIQN,
				sd.LunID,
			}
			dataRow = append(dataRow, newData...)
		}
	}

	data := [][]interface{}{dataRow}

	topRow := fmt.Sprintf("Instance %s",
		instance.InstanceSubdomainPermanent,
	)

	return renderTransposedTable("Records", topRow, getStringParam(c.Arguments["format"]), data, schema)
}

func getIPsAsStringArray(ips []metalcloud.IP) []string {
	sList := []string{}
	for _, ip := range ips {
		sList = append(sList, ip.IPHumanReadable)
	}
	return sList
}
