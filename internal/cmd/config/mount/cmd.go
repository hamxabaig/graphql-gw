package new

import (
	"github.com/chirino/graphql-gw/internal/cmd/config"
	"github.com/chirino/graphql-gw/internal/cmd/root"
	"github.com/chirino/graphql-gw/internal/gateway"
	"github.com/chirino/graphql/schema"
	"github.com/spf13/cobra"
)

var (
	Command = &cobra.Command{
		Use:   "mount [upstream] [type]",
		Short: "mount an upstream into the gateway schema",
		Args:  cobra.ExactArgs(2),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			upstream = args[0]
			schemaType = args[1]
			return config.PreRunLoad(cmd, args)
		},
		Run: run,
	}
	upstream    string
	query       string
	schemaType  string
	field       string
	description string
)

func init() {
	Command.Flags().StringVar(&query, "query", "query {}", "a partial graphql query what the root path to mount from the upstream server")
	Command.Flags().StringVar(&field, "field", "", "field name to mount onto, if none, then all child fields of the query get mounted")
	Command.Flags().StringVar(&description, "description", "", "description to add to the field (shown when introspected)")
	config.Command.AddCommand(Command)
}

func run(_ *cobra.Command, _ []string) {

	c := config.Value
	log := c.Log

	if _, ok := c.Config.Upstreams[upstream]; !ok {
		log.Fatalf("upstream %s not found in the configuration", upstream)
	}

	document := schema.QueryDocument{}
	err := document.ParseWithDescriptions(query)
	if err != nil {
		log.Fatalf("invalid query argument: "+root.Verbosity, err)
	}

	gw, err := gateway.New(c.Config)
	if err != nil {
		log.Fatalf("existing gateway configuration is invalid: "+root.Verbosity, err)
	}

	if gw.Schema.Types[schemaType] == nil {
		log.Fatalf("gateway does not curretly have type named: %s", schemaType)
	}

	byName := map[string]*gateway.TypeConfig{}
	for _, t := range c.Types {
		existing := byName[t.Name]
		if existing != nil {
			existing.Actions = append(existing.Actions, t.Actions...)
		} else {
			byName[t.Name] = &t
		}
	}

	existing := byName[schemaType]
	if existing == nil {
		existing = &gateway.TypeConfig{Name: schemaType}
		byName[schemaType] = existing
	}

	existing.Actions = append(existing.Actions, gateway.ActionWrapper{
		Action: &gateway.Mount{
			Field:       field,
			Description: description,
			Upstream:    upstream,
			Query:       query,
		},
	})

	c.Types = []gateway.TypeConfig{}
	for _, t := range byName {
		c.Types = append(c.Types, *t)
	}

	err = config.Store(*c)
	if err != nil {
		log.Fatalf(root.Verbosity, err)
	}
	log.Printf(`mount added`)
}
