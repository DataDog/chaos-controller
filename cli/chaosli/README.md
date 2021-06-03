# ChaosCLI

The ChaosCLI is motivated by creating a more user friendly and digestible chaos-controller experience. The CLI pairs with your Disruption configuration, giving you information to give you a better sense of what is going on or what will happen when you apply new disruptions. With features like explain and validation, it gives users, not only a better understanding, but a better experience with the controller.  


#### Table of Contents
---
- Validate
- Explain
- Create

#### Validate
---
Usage: `go run chaosli/main.go validate --path <path to disruption file>`

Description: Validates your disruption files to make sure everything is correctly formatted.

#### Explain
---
Usage: `go run chaosli/main.go explain --path <path to disruption file>`

Description: Prints out a summary of the disruption defined by `path`.

#### Creation
---
Usage: `go run chaosli/main.go create --path <path of new disruption file>`

Description: User friendly input process that helps you create your disruptions from scratch answering simple questions.
