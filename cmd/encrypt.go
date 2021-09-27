package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/wantedly/valec/aws"
	"github.com/wantedly/valec/secret"
	"github.com/wantedly/valec/util"
)

// encryptCmd represents the encrypt command
var encryptCmd = &cobra.Command{
	Use:   "encrypt [KEY1=VALUE1 [KEY2=VALUE2 ...]] [-] [-i KEY1 [KEY2 ...]]",
	Short: "Encrypt secret",
	Long: `Encrypt secret

Read from command line arguments:
  $ valec encrypt KEY1=VALUE1 KEY2=VALUE2

Read from stdin:
  $ cat .env
  KEY1=VALUE1
  KEY2=VALUE2
  $ cat .env | valec encrypt -

Enter secret value interactively:
  $ valec encrypt -i KEY1 KEY2
  KEY1:
  KEY2:
`,
	RunE: doEncrypt,
}

var encryptOpts = struct {
	interactive bool
	kmsKey      string
	secretFile  string
}{}

func doEncrypt(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return errors.New("Please specify KEY=VALUE.")
	}

	secretMap := map[string]string{}
	var err error

	if args[0] == "-" {
		secretMap, err = readFromStdin(encryptOpts.kmsKey)
		if err != nil {
			return errors.Wrap(err, "Failed to read secret from stdin.")
		}
	} else {
		if encryptOpts.interactive {
			fmt.Println("Entered secret value will be hidden.")
			secretMap, err = readFromArgsInteractive(args, encryptOpts.kmsKey)
			if err != nil {
				return errors.Wrap(err, "Failed to read secret from args.")
			}
		} else {
			secretMap, err = readFromArgs(args, encryptOpts.kmsKey)
			if err != nil {
				return errors.Wrap(err, "Failed to read secret from args.")
			}
		}
	}

	if encryptOpts.secretFile == "" {
		flushToStdout(secretMap)
	} else {
		if err := flushToFile(secretMap, encryptOpts.secretFile, encryptOpts.kmsKey); err != nil {
			return errors.Wrapf(err, "Failed to flush secrets to file. filename=%s", encryptOpts.secretFile)
		}
	}

	return nil
}

func flushToFile(secretMap map[string]string, filename, kmsKey string) error {
	newSecretMap := map[string]string{}

	if _, err := os.Stat(filename); err == nil {
		currentKmsKey, secrets, err2 := secret.LoadFromYAML(filename)
		if err2 != nil {
			return errors.Wrapf(err2, "Failed to load local secret file. filename=%s", filename)
		}

		if kmsKey != currentKmsKey {
			return errors.Errorf("KMS key alias does not match. current: %s, given: %s", currentKmsKey, kmsKey)
		}

		newSecretMap = secrets.ListToMap()
	}

	for k, v := range secretMap {
		newSecretMap[k] = v
	}
	newSecrets := secret.MapToList(newSecretMap)

	if err := newSecrets.SaveAsYAML(filename, kmsKey); err != nil {
		return errors.Wrapf(err, "Failed to update local secret file. filename=%s", filename)
	}

	return nil
}

func flushToStdout(secretMap map[string]string) {
	for _, v := range secretMap {
		fmt.Println(v)
	}
}

func readFromStdin(kmsKey string) (map[string]string, error) {
	secretMap := map[string]string{}
	lines := util.ScanLines(os.Stdin)

	for _, line := range lines {
		ss := strings.SplitN(line, "=", 2)
		if len(ss) < 2 {
			continue
		}
		key, value := ss[0], ss[1]

		cipherText, err := aws.KMS.EncryptBase64(kmsKey, key, value)
		if err != nil {
			return map[string]string{}, errors.Wrapf(err, "Failed to encrypt secret. key=%s", key)
		}

		secretMap[key] = cipherText
	}

	return secretMap, nil
}

func readFromArgs(args []string, kmsKey string) (map[string]string, error) {
	secretMap := map[string]string{}

	for _, arg := range args {
		ss := strings.SplitN(arg, "=", 2)
		if len(ss) < 2 {
			return map[string]string{}, errors.Errorf("Given argument is invalid format, should be KEY=VALUE. argument=%q", arg)
		}
		key, value := ss[0], ss[1]

		cipherText, err := aws.KMS.EncryptBase64(encryptOpts.kmsKey, key, value)
		if err != nil {
			return map[string]string{}, errors.Wrapf(err, "Failed to encrypt secret. key=%s", key)
		}

		secretMap[key] = cipherText
	}

	return secretMap, nil
}

func readFromArgsInteractive(args []string, kmsKey string) (map[string]string, error) {
	secretMap := map[string]string{}

	for _, arg := range args {
		key := arg
		value := util.ScanNoecho(key)

		cipherText, err := aws.KMS.EncryptBase64(kmsKey, key, value)
		if err != nil {
			return map[string]string{}, errors.Wrapf(err, "Failed to encrypt secret. key=%s", key)
		}

		secretMap[key] = cipherText
	}

	return secretMap, nil
}

func init() {
	RootCmd.AddCommand(encryptCmd)

	encryptCmd.Flags().StringVar(&encryptOpts.secretFile, "add", "", "Add to local secret file")
	encryptCmd.Flags().BoolVarP(&encryptOpts.interactive, "interactive", "i", false, "Interactive value input")
	encryptCmd.Flags().StringVarP(&encryptOpts.kmsKey, "key", "k", secret.DefaultKMSKey, "KMS key alias")
}
