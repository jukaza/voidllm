package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/voidmind-io/voidllm/internal/config"
	"github.com/voidmind-io/voidllm/internal/db"
	"github.com/voidmind-io/voidllm/pkg/crypto"
)

// runMigrateProviderData migrates legacy provider keys and model deployments into
// the new provider_connections, provider_upstream_models, and model_route_steps tables.
func runMigrateProviderData(args []string) {
	fs := flag.NewFlagSet("migrate-provider-data", flag.ExitOnError)
	dsn := fs.String("dsn", "", "Database DSN (defaults to config file)")
	configPath := fs.String("config", "", "path to voidllm.yaml")
	dryRun := fs.Bool("dry-run", false, "Report actions without writing")
	fs.Parse(args) //nolint:errcheck

	ctx := context.Background()
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	var dbCfg config.DatabaseConfig
	var encKey []byte

	if *dsn != "" {
		dbCfg = config.DatabaseConfig{Driver: detectDriver(*dsn), DSN: cleanDSN(*dsn)}
	} else {
		cfg, _, err := config.Load(*configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "load config: %v\n", err)
			os.Exit(1)
		}
		dbCfg = cfg.Database
		var perr error
		encKey, perr = crypto.ParseKey(cfg.Settings.EncryptionKey)
		if perr != nil {
			fmt.Fprintf(os.Stderr, "parse encryption key: %v\n", perr)
			os.Exit(1)
		}
	}

	database, err := db.Open(ctx, dbCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open database: %v\n", err)
		os.Exit(1)
	}
	defer database.Close() //nolint:errcheck

	if err := db.RunMigrations(ctx, database.SQL(), database.Dialect(), log); err != nil {
		fmt.Fprintf(os.Stderr, "run migrations: %v\n", err)
		os.Exit(1)
	}

	provs, err := database.ListProviders(ctx, "", 1000)
	if err != nil {
		fmt.Fprintf(os.Stderr, "list providers: %v\n", err)
		os.Exit(1)
	}

	connCreated := 0
	upstreamCreated := 0
	routesCreated := 0

	for _, prov := range provs {
		conns, _ := database.ListProviderConnections(ctx, prov.ID, false)
		if len(conns) == 0 && prov.APIKeyEncrypted != nil && encKey != nil {
			plain, decErr := crypto.DecryptString(*prov.APIKeyEncrypted, encKey, []byte("provider:"+prov.ID))
			if decErr == nil && plain != "" {
				log.Info("migrate provider key → connection", "provider", prov.Name)
				if !*dryRun {
					enc, encErr := crypto.EncryptString(plain, encKey, []byte("provider_connection:pending"))
					if encErr == nil {
						conn, createErr := database.CreateProviderConnection(ctx, db.CreateProviderConnectionParams{
							ProviderID: prov.ID, Name: "Migrated Primary", AuthType: "apikey",
							Priority: 1, IsActive: true, APIKeyEncrypted: &enc,
						})
						if createErr == nil {
							reEnc, _ := crypto.EncryptString(plain, encKey, []byte("provider_connection:"+conn.ID))
							_, _ = database.UpdateProviderConnection(ctx, conn.ID, db.UpdateProviderConnectionParams{
								APIKeyEncrypted: &reEnc,
							})
							connCreated++
						}
					}
				} else {
					connCreated++
				}
			}
		}

		models, _ := database.ListModels(ctx, "", 500, true)
		for _, m := range models {
			deps, _ := database.ListDeployments(ctx, m.ID)
			if len(deps) == 0 {
				continue
			}
			steps, _ := database.ListModelRouteSteps(ctx, m.ID, false)
			if len(steps) > 0 {
				continue
			}

			var inputs []db.ModelRouteStepInput
			for _, dep := range deps {
				if dep.ProviderID == nil {
					continue
				}
				upstreamID := dep.UpstreamModel
				if upstreamID == "" {
					upstreamID = m.Name
				}
				log.Info("migrate deployment → upstream + route",
					"model", m.Name, "provider", *dep.ProviderID, "upstream", upstreamID)
				if !*dryRun {
					_, _ = database.UpsertProviderUpstreamModel(ctx, db.UpsertProviderUpstreamModelParams{
						ProviderID: *dep.ProviderID, UpstreamID: upstreamID, IsEnabled: dep.IsActive,
						CostInputPer1M: dep.CostInputPer1M, CostOutputPer1M: dep.CostOutputPer1M,
					})
					upstreamCreated++
				} else {
					upstreamCreated++
				}
				inputs = append(inputs, db.ModelRouteStepInput{
					ProviderID: *dep.ProviderID, UpstreamModel: upstreamID, IsEnabled: dep.IsActive,
				})
			}
			if len(inputs) > 0 {
				if !*dryRun {
					if _, err := database.ReplaceModelRouteSteps(ctx, m.ID, inputs); err == nil {
						routesCreated++
					}
				} else {
					routesCreated++
				}
			}
		}
	}

	fmt.Printf("migrate-provider-data complete (dry_run=%v)\n", *dryRun)
	fmt.Printf("  connections created: %d\n", connCreated)
	fmt.Printf("  upstream models upserted: %d\n", upstreamCreated)
	fmt.Printf("  model route steps: %d\n", routesCreated)
}