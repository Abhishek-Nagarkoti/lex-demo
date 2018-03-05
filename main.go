package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lexmodelbuildingservice"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"net/http"
	"os"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}
	r := gin.Default()
	r.POST("/", createBot)
	r.PUT("/", updateBot)
	r.Run()
}

//create a new bot
func createBot(ctx *gin.Context) {
	body := struct {
		Name                 string   `json:"name"`
		ChildDirected        bool     `json:"child_directed"`
		Locale               string   `json:"locale"`
		AbortMessages        []string `json:"abort_messages"`
		ClarificationPrompts []string `json:"clarification_prompts"`
	}{}
	if err := ctx.Bind(&body); err != nil { //validation error
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": "Validation Error."})
	} else {
		cred := credentials.NewStaticCredentials(os.Getenv("ACCESS_KEY_ID"), os.Getenv("SECRET_ACCESS_KEY"), "")
		config := aws.NewConfig().WithCredentials(cred).WithRegion(os.Getenv("AWS_REGION"))
		sess := session.Must(session.NewSession(config))
		svc := lexmodelbuildingservice.New(sess)
		var clarificationPrompts []*lexmodelbuildingservice.Message
		for _, val := range body.ClarificationPrompts {
			clarificationPrompts = append(clarificationPrompts, &lexmodelbuildingservice.Message{
				Content:     aws.String(val),
				ContentType: aws.String("PlainText"),
			})
		}
		var abortMessages []*lexmodelbuildingservice.Message
		for _, val := range body.AbortMessages {
			abortMessages = append(abortMessages, &lexmodelbuildingservice.Message{
				Content:     aws.String(val),
				ContentType: aws.String("PlainText"),
			})
		}
		_, err = svc.PutBot(&lexmodelbuildingservice.PutBotInput{
			Name:                aws.String(body.Name),
			ChildDirected:       aws.Bool(body.ChildDirected),
			Locale:              aws.String(body.Locale),
			ClarificationPrompt: &lexmodelbuildingservice.Prompt{Messages: clarificationPrompts, MaxAttempts: aws.Int64(5)},
			AbortStatement:      &lexmodelbuildingservice.Statement{Messages: abortMessages},
		})
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "message": "Server Error."})
		} else {
			_, err := svc.PutBotAlias(&lexmodelbuildingservice.PutBotAliasInput{
				BotName:    aws.String(body.Name),
				BotVersion: aws.String("$LATEST"),
				Name:       aws.String(body.Name),
			})
			if err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "message": "Server Error."})
			} else {
				ctx.JSON(http.StatusOK, gin.H{"error": nil, "message": "New Bot Created."})
			}
		}
	}
}

//create new intent and link it with existing bot
func updateBot(ctx *gin.Context) {
	body := struct {
		Name                 string   `json:"name"`
		ChildDirected        bool     `json:"child_directed"`
		Locale               string   `json:"locale"`
		Messages             []string `json:"messages"`
		Utterances           []string `json:"utterances"`
		IntentName           string   `json:"intent_name"`
		AbortMessages        []string `json:"abort_messages"`
		ClarificationPrompts []string `json:"clarification_prompts"`
		Version              string   `json:"version"`
	}{}
	if err := ctx.Bind(&body); err != nil { //validation error
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": "Validation Error."})
	} else {
		cred := credentials.NewStaticCredentials(os.Getenv("ACCESS_KEY_ID"), os.Getenv("SECRET_ACCESS_KEY"), "")
		config := aws.NewConfig().WithCredentials(cred).WithRegion(os.Getenv("AWS_REGION"))
		sess := session.Must(session.NewSession(config))
		svc := lexmodelbuildingservice.New(sess)
		input := &lexmodelbuildingservice.GetBotInput{
			Name:           aws.String(body.Name),
			VersionOrAlias: aws.String(body.Version), //use "$LATEST" for latest version
		}
		bot, err := svc.GetBot(input)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "message": "Server Error."})
		} else {
			var messages []*lexmodelbuildingservice.Message
			for _, val := range body.Messages {
				messages = append(messages, &lexmodelbuildingservice.Message{
					Content:     aws.String(val),
					ContentType: aws.String("PlainText"),
				})
			}
			var utterances []*string
			for _, val := range body.Utterances {
				utterances = append(utterances, aws.String(val))
			}
			intent := &lexmodelbuildingservice.PutIntentInput{
				Name: aws.String(body.IntentName),
				ConclusionStatement: &lexmodelbuildingservice.Statement{
					Messages: messages,
				},
				SampleUtterances:    utterances,
				FulfillmentActivity: &lexmodelbuildingservice.FulfillmentActivity{Type: aws.String("ReturnIntent")},
			}
			result, err := svc.PutIntent(intent)
			if err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "message": "Server Error."})
			} else {
				bot.Intents = append(bot.Intents, &lexmodelbuildingservice.Intent{IntentName: result.Name, IntentVersion: result.Version})
				var clarificationPrompts []*lexmodelbuildingservice.Message
				for _, val := range body.ClarificationPrompts {
					clarificationPrompts = append(clarificationPrompts, &lexmodelbuildingservice.Message{
						Content:     aws.String(val),
						ContentType: aws.String("PlainText"),
					})
				}
				var abortMessages []*lexmodelbuildingservice.Message
				for _, val := range body.AbortMessages {
					abortMessages = append(abortMessages, &lexmodelbuildingservice.Message{
						Content:     aws.String(val),
						ContentType: aws.String("PlainText"),
					})
				}
				_, err = svc.PutBot(&lexmodelbuildingservice.PutBotInput{
					Checksum:            bot.Checksum,
					Intents:             bot.Intents,
					Name:                bot.Name,
					ChildDirected:       bot.ChildDirected,
					Locale:              bot.Locale,
					ClarificationPrompt: &lexmodelbuildingservice.Prompt{Messages: clarificationPrompts, MaxAttempts: aws.Int64(5)},
					AbortStatement:      &lexmodelbuildingservice.Statement{Messages: abortMessages},
				})
				if err != nil {
					ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "message": "Server Error."})
				} else {
					ctx.JSON(http.StatusOK, gin.H{"error": nil, "message": "Bot Updated with new intent."})
				}
			}
		}
	}
}
