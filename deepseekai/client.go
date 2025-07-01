package deepseekai

import (
	"context"
	"fmt"

	"github.com/cohesion-org/deepseek-go"
)

const Prompt string = "You are an assistant that converts natural language schedules into valid 5-field cron" +
	" expressions in UTC: Minutes, Hours, Day of Month, Month, Day of Week. Fields accept *, /, ,, and -; ? is allowed only in Day of Month and Day of Week. Minutes: 0–59, Hours: 0–23, Day of Month: 1–31, Month: 1–12 or JAN–DEC, Day of Week: 0–6 or SUN–SAT (Sunday is 0). The smallest allowed interval is 1 minute (cron does not support seconds). If no timezone is provided, ask for the user's country to convert to UTC. Confirm the schedule only in natural language, never show the cron expression. Once confirmed, respond only with “final cron is <cron expression>” and nothing else. If the input is invalid, reply that the schedule is unsupported. In all cases, continue prompting the user for a valid natural language schedule and timezone until a valid and confirmed cron expression is produced. Keep all responses minimal and precise."

type Client struct {
	client *deepseek.Client
}

func NewClient(apiKey string) *Client {
	return &Client{client: deepseek.NewClient(apiKey)}
}

func (c *Client) Converse(messages []deepseek.ChatCompletionMessage) (*deepseek.Message, error) {
	request := &deepseek.ChatCompletionRequest{
		Model:    deepseek.DeepSeekChat,
		Messages: messages,
	}
	response, err := c.client.CreateChatCompletion(context.Background(), request)
	if err != nil {
		return nil, fmt.Errorf("failed to get response from deepseek [messages: %+v]: %w", messages, err)
	}
	return &response.Choices[0].Message, nil
}
