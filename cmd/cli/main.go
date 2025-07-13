package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/glizzus/sound-off/internal/config"
	"github.com/glizzus/sound-off/internal/datalayer"
	"github.com/glizzus/sound-off/internal/generator"
	"github.com/glizzus/sound-off/internal/repository"
	"github.com/urfave/cli/v2"
)

var stdinReader = bufio.NewReader(os.Stdin)

var uuidGenerator = generator.UUIDV4Generator{}

func prompt(label string) string {
	fmt.Printf("%s: ", label)
	input, _ := stdinReader.ReadString('\n')
	return strings.TrimSpace(input)
}

func main() {
	if err := config.LoadEnv(); err != nil {
		log.Fatalf("Failed to load .env file: %v", err)
	}

	pool, err := datalayer.NewPostgresPoolFromEnv()
	if err != nil {
		log.Fatalf("Failed to create postgres pool: %v", err)
	}
	if err := datalayer.MigratePostgres(pool); err != nil {
		log.Fatalf("Failed to migrate postgres: %v", err)
	}
	repo := repository.NewPostgresSoundCronRepository(pool)

	app := &cli.App{
		Name:        "sound-off-cli",
		Description: "A development CLI tool for testing Sound Off without Discord",
		Commands: []*cli.Command{
			{
				Name:  "list",
				Usage: "List all upcoming jobs for a specific guild",
				Action: func(c *cli.Context) error {
					guildID := c.String("guild-id")
					if guildID == "" {
						return cli.Exit("Please provide a guild ID using --guild-id", 1)
					}

					jobs, err := repo.List(c.Context, guildID)
					if err != nil {
						return cli.Exit("Failed to retrieve jobs: "+err.Error(), 1)
					}

					if len(jobs) == 0 {
						log.Println("No upcoming jobs found for the specified guild.")
						return nil
					}

					for _, job := range jobs {
						log.Printf("%+v", job)
					}
					return nil
				},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "guild-id",
						Usage:    "ID of the guild to list jobs for",
						Required: true,
					},
				},
			},
			{
				Name:  "add",
				Usage: "Add a new job to the repository",
				Action: func(c *cli.Context) error {
					guildID := c.String("guild-id")
					if guildID == "" {
						return cli.Exit("Please provide a guild ID using --guild-id", 1)
					}

					name := prompt("Enter job name")
					cron := prompt("Enter cron expression (e.g., '0 0 * * *')")
					fileSizeStr := prompt("Enter file size in bytes (e.g., '1048576')")
					fileSize, err := strconv.ParseInt(fileSizeStr, 10, 64)
					if err != nil {
						return cli.Exit("Invalid file size: "+err.Error(), 1)
					}

					id, _ := uuidGenerator.Next()

					sc := repository.SoundCron{
						ID:       id,
						GuildID:  guildID,
						Name:     name,
						Cron:     cron,
						FileSize: fileSize,
					}

					if err := repo.Save(c.Context, sc); err != nil {
						return cli.Exit("Failed to save job: "+err.Error(), 1)
					}

					log.Println("Job added successfully.")
					return nil
				},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "guild-id",
						Usage:    "ID of the guild to add the job for",
						Required: true,
					},
				},
			},
		},
	}

	err = app.Run(os.Args)
	if err != nil {
		log.Fatalf("Error running CLI: %v", err)
	}
}
