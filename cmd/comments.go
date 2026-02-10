package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	hbapi "github.com/honeybadger-io/api-go"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	commentsProjectID    int
	commentsFaultID      int
	commentID            int
	commentsOutputFormat string
	commentBody          string
)

// commentsCmd represents the comments command
var commentsCmd = &cobra.Command{
	Use:     "comments",
	Short:   "Manage fault comments",
	GroupID: GroupDataAPI,
	Long:    `View and manage comments on faults in your Honeybadger projects.`,
}

// commentsListCmd represents the comments list command
var commentsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List comments for a fault",
	Long:  `List all comments on a specific fault.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if commentsProjectID == 0 {
			commentsProjectID = viper.GetInt("project_id")
		}
		if commentsProjectID == 0 {
			return fmt.Errorf("project ID is required. Set it using --project-id flag or HONEYBADGER_PROJECT_ID environment variable")
		}
		if commentsFaultID == 0 {
			return fmt.Errorf("fault ID is required. Set it using --fault-id flag")
		}

		authToken := viper.GetString("auth_token")
		if authToken == "" {
			return fmt.Errorf(
				"auth token is required. Set it using --auth-token flag or HONEYBADGER_AUTH_TOKEN environment variable",
			)
		}

		endpoint := convertEndpointForDataAPI(viper.GetString("endpoint"))

		client := hbapi.NewClient().
			WithBaseURL(endpoint).
			WithAuthToken(authToken)

		ctx := context.Background()
		comments, err := client.Comments.List(ctx, commentsProjectID, commentsFaultID)
		if err != nil {
			return fmt.Errorf("failed to list comments: %w", err)
		}

		switch commentsOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(comments, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "ID\tAUTHOR\tEVENT\tCREATED\tBODY")
			for _, c := range comments {
				author := "System"
				if c.Author != "" {
					author = c.Author
				}

				body := c.Body
				if len(body) > 40 {
					body = body[:37] + "..."
				}

				_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n",
					c.ID,
					author,
					c.Event,
					c.CreatedAt.Format("2006-01-02 15:04"),
					body)
			}
			_ = w.Flush()
		}

		return nil
	},
}

// commentsGetCmd represents the comments get command
var commentsGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a comment by ID",
	Long:  `Get detailed information about a specific comment.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if commentsProjectID == 0 {
			commentsProjectID = viper.GetInt("project_id")
		}
		if commentsProjectID == 0 {
			return fmt.Errorf("project ID is required. Set it using --project-id flag or HONEYBADGER_PROJECT_ID environment variable")
		}
		if commentsFaultID == 0 {
			return fmt.Errorf("fault ID is required. Set it using --fault-id flag")
		}
		if commentID == 0 {
			return fmt.Errorf("comment ID is required. Set it using --id flag")
		}

		authToken := viper.GetString("auth_token")
		if authToken == "" {
			return fmt.Errorf(
				"auth token is required. Set it using --auth-token flag or HONEYBADGER_AUTH_TOKEN environment variable",
			)
		}

		endpoint := convertEndpointForDataAPI(viper.GetString("endpoint"))

		client := hbapi.NewClient().
			WithBaseURL(endpoint).
			WithAuthToken(authToken)

		ctx := context.Background()
		comment, err := client.Comments.Get(ctx, commentsProjectID, commentsFaultID, commentID)
		if err != nil {
			return fmt.Errorf("failed to get comment: %w", err)
		}

		switch commentsOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(comment, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			fmt.Printf("Comment Details:\n")
			fmt.Printf("  ID: %d\n", comment.ID)
			fmt.Printf("  Fault ID: %d\n", comment.FaultID)
			fmt.Printf("  Event: %s\n", comment.Event)
			fmt.Printf("  Source: %s\n", comment.Source)
			if comment.Author != "" {
				fmt.Printf("  Author: %s\n", comment.Author)
			}
			fmt.Printf("  Created: %s\n", comment.CreatedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("  Body:\n    %s\n", comment.Body)
		}

		return nil
	},
}

// commentsCreateCmd represents the comments create command
var commentsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new comment",
	Long:  `Create a new comment on a fault.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if commentsProjectID == 0 {
			commentsProjectID = viper.GetInt("project_id")
		}
		if commentsProjectID == 0 {
			return fmt.Errorf("project ID is required. Set it using --project-id flag or HONEYBADGER_PROJECT_ID environment variable")
		}
		if commentsFaultID == 0 {
			return fmt.Errorf("fault ID is required. Set it using --fault-id flag")
		}
		if commentBody == "" {
			return fmt.Errorf("comment body is required. Set it using --body flag")
		}

		authToken := viper.GetString("auth_token")
		if authToken == "" {
			return fmt.Errorf(
				"auth token is required. Set it using --auth-token flag or HONEYBADGER_AUTH_TOKEN environment variable",
			)
		}

		endpoint := convertEndpointForDataAPI(viper.GetString("endpoint"))

		client := hbapi.NewClient().
			WithBaseURL(endpoint).
			WithAuthToken(authToken)

		ctx := context.Background()
		comment, err := client.Comments.Create(ctx, commentsProjectID, commentsFaultID, commentBody)
		if err != nil {
			return fmt.Errorf("failed to create comment: %w", err)
		}

		switch commentsOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(comment, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			fmt.Printf("Comment created successfully!\n")
			fmt.Printf("  ID: %d\n", comment.ID)
			fmt.Printf("  Body: %s\n", comment.Body)
		}

		return nil
	},
}

// commentsUpdateCmd represents the comments update command
var commentsUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update an existing comment",
	Long:  `Update the body of an existing comment.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if commentsProjectID == 0 {
			commentsProjectID = viper.GetInt("project_id")
		}
		if commentsProjectID == 0 {
			return fmt.Errorf("project ID is required. Set it using --project-id flag or HONEYBADGER_PROJECT_ID environment variable")
		}
		if commentsFaultID == 0 {
			return fmt.Errorf("fault ID is required. Set it using --fault-id flag")
		}
		if commentID == 0 {
			return fmt.Errorf("comment ID is required. Set it using --id flag")
		}
		if commentBody == "" {
			return fmt.Errorf("comment body is required. Set it using --body flag")
		}

		authToken := viper.GetString("auth_token")
		if authToken == "" {
			return fmt.Errorf(
				"auth token is required. Set it using --auth-token flag or HONEYBADGER_AUTH_TOKEN environment variable",
			)
		}

		endpoint := convertEndpointForDataAPI(viper.GetString("endpoint"))

		client := hbapi.NewClient().
			WithBaseURL(endpoint).
			WithAuthToken(authToken)

		ctx := context.Background()
		if err := client.Comments.Update(
			ctx,
			commentsProjectID,
			commentsFaultID,
			commentID,
			commentBody,
		); err != nil {
			return fmt.Errorf("failed to update comment: %w", err)
		}

		comment, err := client.Comments.Get(ctx, commentsProjectID, commentsFaultID, commentID)
		if err != nil {
			return fmt.Errorf("failed to fetch updated comment: %w", err)
		}

		switch commentsOutputFormat {
		case "json":
			jsonBytes, err := json.MarshalIndent(comment, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
		default:
			fmt.Printf("Comment updated successfully!\n")
			fmt.Printf("  ID: %d\n", comment.ID)
			fmt.Printf("  Body: %s\n", comment.Body)
		}

		return nil
	},
}

// commentsDeleteCmd represents the comments delete command
var commentsDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a comment",
	Long:  `Delete a comment by ID. This action cannot be undone.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if commentsProjectID == 0 {
			commentsProjectID = viper.GetInt("project_id")
		}
		if commentsProjectID == 0 {
			return fmt.Errorf("project ID is required. Set it using --project-id flag or HONEYBADGER_PROJECT_ID environment variable")
		}
		if commentsFaultID == 0 {
			return fmt.Errorf("fault ID is required. Set it using --fault-id flag")
		}
		if commentID == 0 {
			return fmt.Errorf("comment ID is required. Set it using --id flag")
		}

		authToken := viper.GetString("auth_token")
		if authToken == "" {
			return fmt.Errorf(
				"auth token is required. Set it using --auth-token flag or HONEYBADGER_AUTH_TOKEN environment variable",
			)
		}

		endpoint := convertEndpointForDataAPI(viper.GetString("endpoint"))

		client := hbapi.NewClient().
			WithBaseURL(endpoint).
			WithAuthToken(authToken)

		ctx := context.Background()
		err := client.Comments.Delete(ctx, commentsProjectID, commentsFaultID, commentID)
		if err != nil {
			return fmt.Errorf("failed to delete comment: %w", err)
		}

		fmt.Println("Comment deleted successfully")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(commentsCmd)
	commentsCmd.AddCommand(commentsListCmd)
	commentsCmd.AddCommand(commentsGetCmd)
	commentsCmd.AddCommand(commentsCreateCmd)
	commentsCmd.AddCommand(commentsUpdateCmd)
	commentsCmd.AddCommand(commentsDeleteCmd)

	// Common flags
	commentsCmd.PersistentFlags().IntVar(&commentsProjectID, "project-id", 0, "Project ID")
	commentsCmd.PersistentFlags().IntVar(&commentsFaultID, "fault-id", 0, "Fault ID")

	// Flags for list command
	commentsListCmd.Flags().
		StringVarP(&commentsOutputFormat, "output", "o", "table", "Output format (table or json)")

	// Flags for get command
	commentsGetCmd.Flags().IntVar(&commentID, "id", 0, "Comment ID")
	commentsGetCmd.Flags().
		StringVarP(&commentsOutputFormat, "output", "o", "text", "Output format (text or json)")
	_ = commentsGetCmd.MarkFlagRequired("id")

	// Flags for create command
	commentsCreateCmd.Flags().StringVar(&commentBody, "body", "", "Comment body text")
	commentsCreateCmd.Flags().
		StringVarP(&commentsOutputFormat, "output", "o", "text", "Output format (text or json)")
	_ = commentsCreateCmd.MarkFlagRequired("body")

	// Flags for update command
	commentsUpdateCmd.Flags().IntVar(&commentID, "id", 0, "Comment ID")
	commentsUpdateCmd.Flags().StringVar(&commentBody, "body", "", "New comment body text")
	commentsUpdateCmd.Flags().
		StringVarP(&commentsOutputFormat, "output", "o", "text", "Output format (text or json)")
	_ = commentsUpdateCmd.MarkFlagRequired("id")
	_ = commentsUpdateCmd.MarkFlagRequired("body")

	// Flags for delete command
	commentsDeleteCmd.Flags().IntVar(&commentID, "id", 0, "Comment ID")
	_ = commentsDeleteCmd.MarkFlagRequired("id")
}
