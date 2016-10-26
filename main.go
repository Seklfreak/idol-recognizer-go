package main

import (
	"./lib"

	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/urfave/cli"
	"gopkg.in/ini.v1"
)

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
		!cfg.Section("faceplusplus").HasKey("api url") {
		cfg.Section("faceplusplus").NewKey("api key", "yourapikey")
		cfg.Section("faceplusplus").NewKey("api secret", "yourapisecret")
		cfg.Section("faceplusplus").NewKey("api url", "https://apius.faceplusplus.com")
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

	app.Commands = []cli.Command{
		{
			Name:  "person",
			Usage: "manage persons",
			Subcommands: []cli.Command{
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
						var err error
						detectedFace := new(facepp.DetectionDetectO)
						if imageUrl != "" {
							detectedFace, err = fpp.DetectionDetect(imageUrl, "", "")
							if err != nil {
								return cli.NewExitError(err.Error(), 1)
							}
						} else if imageFileLocation != "" {
							detectedFace, err = fpp.DetectionDetectFile(imageFileLocation, "", "")
							if err != nil {
								return cli.NewExitError(err.Error(), 1)
							}
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
						var err error
						faceInfo := new(facepp.RecognitionIdentifyO)
						if imageUrl != "" {
							faceInfo, err = fpp.RecognitionIdentify(c.Args().Get(0), imageUrl, "oneface", "")
							if err != nil {
								return cli.NewExitError(err.Error(), 1)
							}
						} else if imageFileLocation != "" {
							faceInfo, err = fpp.RecognitionIdentifyFile(c.Args().Get(0), imageFileLocation, "oneface", "")
							if err != nil {
								return cli.NewExitError(err.Error(), 1)
							}
						}
						if len(faceInfo.Face) <= 0 || len(faceInfo.Face[0].Candidate) <= 0 {
							return cli.NewExitError("found no matching face", 1)
						}
						for _, candidate := range faceInfo.Face[0].Candidate {
							fmt.Println("confidence", strconv.FormatFloat(candidate.Confidence, 'f', 2, 64)+"%:", "found", candidate.Person_name, "("+candidate.Tag+")")
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
