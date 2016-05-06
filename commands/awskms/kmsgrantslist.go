package awskms

import (
	"fmt"

	"github.com/dcoker/biscuit/keymanager"
	"github.com/dcoker/biscuit/shared"
	"github.com/dcoker/biscuit/store"
	"gopkg.in/alecthomas/kingpin.v2"
)

type kmsGrantsList struct {
	name, filename *string
}

// NewKmsGrantsList constructs the command to list grants.
func NewKmsGrantsList(c *kingpin.CmdClause) shared.Command {
	params := &kmsGrantsList{}
	params.name = c.Arg("name", "Name of the secret to list grants for.").Required().String()
	params.filename = shared.FilenameFlag(c)
	return params
}

type grantsForOneAlias struct {
	GranteePrincipal  *string
	RetiringPrincipal *string   `yaml:",omitempty"`
	Operations        []*string `yaml:",flow"`
	GrantIds          map[string]string
}

// Run runs the command.
func (w *kmsGrantsList) Run() error {
	database := store.NewFileStore(*w.filename)
	values, err := database.Get(*w.name)
	if err != nil {
		return err
	}
	values = values.FilterByKeyManager(keymanager.KmsLabel)

	aliases, err := resolveValuesToAliasesAndRegions(values)
	if err != nil {
		return err
	}

	output := make(map[string]map[string]grantsForOneAlias)
	for aliasName, regions := range aliases {
		mrk, err := NewMultiRegionKey(aliasName, regions, "")
		if err != nil {
			return err
		}
		regionGrants, err := mrk.GetGrantDetails()
		if err != nil {
			return err
		}

		// Group by grant name and collect grant IDs into a list by region.
		n2e := make(map[string]grantsForOneAlias)
		for region, grants := range regionGrants {
			for _, grant := range grants {
				if entry, present := n2e[*grant.Name]; present {
					entry.GrantIds[region] = *grant.GrantId
				} else {
					entry := grantsForOneAlias{
						GranteePrincipal:  grant.GranteePrincipal,
						RetiringPrincipal: grant.RetiringPrincipal,
						Operations:        grant.Operations,
					}
					entry.GrantIds = make(map[string]string)
					entry.GrantIds[region] = *grant.GrantId
					n2e[*grant.Name] = entry
				}
			}
		}
		output[aliasName] = n2e
	}
	fmt.Print(shared.MustYaml(output))
	return nil
}