package main

import (
	"./lib"

	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/urfave/cli"
	"gopkg.in/ini.v1"
)

func runAddFaceToPersonFromFile(fpp facepp.Facepp, personName string, fileLocation string) {
	detectedFace, err := fpp.DetectionDetectFile(fileLocation, "", "")
	if err != nil {
		fmt.Println(fileLocation, "failed:", err.Error())
	} else {
		if len(detectedFace.Face) <= 0 {
			fmt.Println(fileLocation, "failed:", "unable to find face in picture")
		} else {
			if len(detectedFace.Face) > 1 {
				fmt.Println(fileLocation, "failed:", "found too many faces in picture (found "+strconv.Itoa(len(detectedFace.Face))+")")
			} else {
				addedFaces, err := fpp.PersonAddFace(personName, detectedFace.Face[0].Face_id)
				if err != nil {
					fmt.Println(fileLocation, "failed:", err.Error())
				} else {
					if addedFaces.Success != true {
						fmt.Println(fileLocation, "failed:", "api error")
					} else {
						fmt.Println(fileLocation, "added", addedFaces.Added, "face to person", personName)
					}
				}
			}
		}
	}
}

func main() {
	var app = cli.NewApp()

	var err error
	cfg, err := ini.Load("config.ini")
	if err != nil {
		fmt.Println("unable to read config file:", err)
		cfg = ini.Empty()
	}

	if !cfg.Section("faceplusplus").HasKey("api key") &&
		!cfg.Section("faceplusplus").HasKey("api secret") &&
		!cfg.Section("faceplusplus").HasKey("api url") &&
		!cfg.Section("faceplusplus").HasKey("concurrent requests") {
		cfg.Section("faceplusplus").NewKey("api key", "yourapikey")
		cfg.Section("faceplusplus").NewKey("api secret", "yourapisecret")
		cfg.Section("faceplusplus").NewKey("api url", "https://apius.faceplusplus.com")
		cfg.Section("faceplusplus").NewKey("concurrent requests", "3")
		err = cfg.SaveTo("config.ini")

		if err != nil {
			fmt.Println("unable to write config file", err)
			return
		}
		fmt.Println("Wrote config file, please fill out and restart the program")
		return
	}

	var fpp = facepp.NewFacepp(cfg.Section("faceplusplus").Key("api url").String(),
		cfg.Section("faceplusplus").Key("api key").String(),
		cfg.Section("faceplusplus").Key("api secret").String())

	var imageUrl string
	var imageFileLocation string
	var block bool
	var tag string

	app.Commands = []cli.Command{
		{
			Name:  "person",
			Usage: "manage persons",
			Subcommands: []cli.Command{
				{
					Name:      "create",
					Usage:     "creates a new person",
					ArgsUsage: "<person name> <group name>",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:        "tag",
							Usage:       "tag of the person you want to add",
							Destination: &tag,
						},
					},
					Action: func(c *cli.Context) error {
						if c.Args().Get(0) == "" || c.Args().Get(1) == "" {
							return cli.NewExitError("not enough arguments", 1)
						}
						newPerson, err := fpp.PersonCreate(c.Args().Get(0), "", tag, c.Args().Get(1))
						if err != nil {
							return cli.NewExitError(err.Error(), 1)
						}
						fmt.Println("created person", newPerson.Person_name, "("+newPerson.Tag+")")
						return nil
					},
				},
				{
					Name:      "add-face",
					Usage:     "adds a new face to a person",
					ArgsUsage: "<person name>",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:        "image-url",
							Usage:       "url of the image with the face you want to add",
							Destination: &imageUrl,
						},
						cli.StringFlag{
							Name:        "image",
							Usage:       "path of the image with the face you want to add",
							Destination: &imageFileLocation,
						},
					},
					Action: func(c *cli.Context) error {
						if c.Args().Get(0) == "" || (imageUrl == "" && imageFileLocation == "") {
							return cli.NewExitError("not enough arguments", 1)
						}
						if imageUrl != "" {
							detectedFace, err := fpp.DetectionDetect(imageUrl, "", "")
							if err != nil {
								return cli.NewExitError(err.Error(), 1)
							}
							if len(detectedFace.Face) <= 0 {
								return cli.NewExitError("unable to find face in picture", 1)
							}
							if len(detectedFace.Face) > 1 {
								return cli.NewExitError("found too many faces in picture (found "+strconv.Itoa(len(detectedFace.Face))+")", 1)
							}
							addedFaces, err := fpp.PersonAddFace(c.Args().Get(0), detectedFace.Face[0].Face_id)
							if err != nil {
								return cli.NewExitError(err.Error(), 1)
							}
							if addedFaces.Success != true {
								return cli.NewExitError("api error", 1)
							}
							fmt.Println("added", addedFaces.Added, "face to person", c.Args().Get(0))
						} else if imageFileLocation != "" {
							pathes, err := filepath.Glob(imageFileLocation)
							if err != nil {
								return cli.NewExitError(err.Error(), 1)
							}

							// source https://gist.github.com/AntoineAugusti/80e99edfe205baf7a094
							maxNbConcurrentGoroutines, err := cfg.Section("faceplusplus").Key("concurrent requests").Int()
							if err != nil {
								return cli.NewExitError(err.Error(), 1)
							}
							concurrentGoroutines := make(chan struct{}, maxNbConcurrentGoroutines)
							for i := 0; i < maxNbConcurrentGoroutines; i++ {
								concurrentGoroutines <- struct{}{}
							}
							done := make(chan bool)
							waitForAllJobs := make(chan bool)

							go func() {
								for i := 0; i < len(pathes); i++ {
									<-done
									concurrentGoroutines <- struct{}{}
								}
								waitForAllJobs <- true
							}()
							for _, path := range pathes {
								<-concurrentGoroutines
								go func(path string) {
									runAddFaceToPersonFromFile(fpp, c.Args().Get(0), path)
									done <- true
								}(path)
							}
							<-waitForAllJobs
						}
						return nil
					},
				},
			},
		},
		{
			Name:  "train",
			Usage: "train the model",
			Subcommands: []cli.Command{
				{
					Name:      "identify",
					Usage:     "trains the identify model for one group",
					ArgsUsage: "<group name>",
					Flags: []cli.Flag{
						cli.BoolTFlag{
							Name:        "block",
							Usage:       "waits until the command is done",
							Destination: &block,
						},
					},
					Action: func(c *cli.Context) error {
						if c.Args().Get(0) == "" {
							return cli.NewExitError("not enough arguments", 1)
						}
						if block == false {
							trainResponse, err := fpp.TrainIdentify(c.Args().Get(0))
							if err != nil {
								return cli.NewExitError(err.Error(), 1)
							}
							fmt.Println("identify model training of group", c.Args().Get(0), "started, your session id:", trainResponse.Session_id)
							fmt.Println("(try info session", trainResponse.Session_id, ")")
						} else {
							trainResponse, err := fpp.TrainIdentify(c.Args().Get(0))
							if err != nil {
								return cli.NewExitError(err.Error(), 1)
							}
							fmt.Println("queued training of the identify model of group", c.Args().Get(0), "(session id:", trainResponse.Session_id, ")")
							sessionInfo := new(facepp.InfoGetSessionO)
							for {
								time.Sleep(1 * time.Second)
								sessionInfo, err = fpp.InfoGetSession(trainResponse.Session_id)
								if err != nil {
									return cli.NewExitError(err.Error(), 1)
								}
								if sessionInfo.Status == "SUCC" {
									fmt.Println("task done")
									break
								} else if sessionInfo.Status == "FAILED" {
									return cli.NewExitError("task failed!", 1)
								} else {
									fmt.Println("status: ", sessionInfo.Status)
								}
							}
						}
						return nil
					},
				},
			},
		},
		{
			Name:  "recognition",
			Usage: "recognize pictures",
			Subcommands: []cli.Command{
				{
					Name:      "identify",
					Usage:     "identify the most similar person within a group (checks the largest face in the given picture)",
					ArgsUsage: "<group name>",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:        "image-url",
							Usage:       "url of the image with the face you want to identify",
							Destination: &imageUrl,
						},
						cli.StringFlag{
							Name:        "image",
							Usage:       "path of the image with the face you want to identify",
							Destination: &imageFileLocation,
						},
					},
					Action: func(c *cli.Context) error {
						if c.Args().Get(0) == "" {
							return cli.NewExitError("not enough arguments", 1)
						}
						if imageUrl != "" {
							faceInfo, err := fpp.RecognitionIdentify(c.Args().Get(0), imageUrl, "oneface", "")
							if err != nil {
								return cli.NewExitError(err.Error(), 1)
							}
							if len(faceInfo.Face) <= 0 || len(faceInfo.Face[0].Candidate) <= 0 {
								return cli.NewExitError("found no matching face", 1)
							}
							for _, candidate := range faceInfo.Face[0].Candidate {
								fmt.Println("confidence", strconv.FormatFloat(candidate.Confidence, 'f', 2, 64)+"%:", "found", candidate.Person_name, "("+candidate.Tag+")")
							}
						} else if imageFileLocation != "" {
							pathes, err := filepath.Glob(imageFileLocation)
							if err != nil {
								return cli.NewExitError(err.Error(), 1)
							}
							for _, path := range pathes {
								faceInfo, err := fpp.RecognitionIdentifyFile(c.Args().Get(0), path, "oneface", "")
								if err != nil {
									fmt.Println(path, "failed:", err.Error())
								} else {
									if len(faceInfo.Face) <= 0 || len(faceInfo.Face[0].Candidate) <= 0 {
										return cli.NewExitError("found no matching face", 1)
									}
									for _, candidate := range faceInfo.Face[0].Candidate {
										fmt.Println(path, "confidence", strconv.FormatFloat(candidate.Confidence, 'f', 2, 64)+"%:", "found", candidate.Person_name, "("+candidate.Tag+")")
									}
								}
							}
						}
						return nil
					},
				},
			},
		},
		{
			Name:  "info",
			Usage: "get information",
			Subcommands: []cli.Command{
				{
					Name:      "session",
					Usage:     "gets the status of an task",
					ArgsUsage: "<session id>",
					Action: func(c *cli.Context) error {
						if c.Args().Get(0) == "" {
							return cli.NewExitError("not enough arguments", 1)
						}
						sessionInfo, err := fpp.InfoGetSession(c.Args().Get(0))
						if err != nil {
							return cli.NewExitError(err.Error(), 1)
						}
						fmt.Println("Status of session", sessionInfo.Session_id, "is", sessionInfo.Status)
						return nil
					},
				},
			},
		},
	}

	app.Run(os.Args)
}
