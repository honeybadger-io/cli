package cmd

import (
	"context"
	"fmt"
	"os"
	"slices"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var (
	commentsProjectID    string
	commentsFaultID      string
	commentID            string
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

// requireCommentFault resolves the project id and checks the fault id every
// comments subcommand needs.
func requireCommentFault() error {
	if err := resolveDataAPIProjectID(&commentsProjectID); err != nil {
		return err
	}
	if commentsFaultID == "" {
		return fmt.Errorf("fault ID is required. Set it using --fault-id flag")
	}
	return nil
}

// commentsListCmd represents the comments list command
var commentsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List comments for a fault",
	Long: `List every comment on a specific fault, oldest first.` + v3Reference(
		"comments/list-comments",
	),
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := requireCommentFault(); err != nil {
			return err
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		comments, err := client.Faults.ListAllComments(
			context.Background(),
			commentsProjectID,
			commentsFaultID,
		)
		if err != nil {
			return dataAPIError("list comments", err)
		}
		// v3 pages newest first; a fault's history reads oldest first, as v2
		// listed it.
		slices.Reverse(comments)

		if commentsOutputFormat == "json" {
			return printJSON(comments)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "ID\tAUTHOR\tEVENT\tCREATED\tBODY")
		for _, c := range comments {
			author := "System"
			if name := c.Author.GetOrEmpty().Name.GetOrEmpty(); name != "" {
				author = name
			}

			body := c.Body.GetOrEmpty()
			if len(body) > 40 {
				body = body[:37] + "..."
			}

			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				c.Id,
				author,
				c.Event.GetOrEmpty(),
				c.CreatedAt.Format("2006-01-02 15:04"),
				body)
		}
		return w.Flush()
	},
}

// commentsGetCmd represents the comments get command
var commentsGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a comment by ID",
	Long: `Get detailed information about a specific comment.` + v3Reference(
		"comments/get-comment",
	),
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := requireCommentFault(); err != nil {
			return err
		}
		if commentID == "" {
			return fmt.Errorf("comment ID is required. Set it using --id flag")
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		comment, err := client.Faults.GetComment(
			context.Background(),
			commentsProjectID,
			commentsFaultID,
			commentID,
		)
		if err != nil {
			return dataAPIError("get comment", err)
		}

		if commentsOutputFormat == "json" {
			return printJSON(comment)
		}

		fmt.Printf("Comment Details:\n")
		fmt.Printf("  ID: %s\n", comment.Id)
		fmt.Printf("  Fault ID: %s\n", comment.FaultId)
		fmt.Printf("  Event: %s\n", comment.Event.GetOrEmpty())
		fmt.Printf("  Source: %s\n", comment.Source)
		if name := comment.Author.GetOrEmpty().Name.GetOrEmpty(); name != "" {
			fmt.Printf("  Author: %s\n", name)
		}
		fmt.Printf("  Created: %s\n", comment.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("  Body:\n    %s\n", comment.Body.GetOrEmpty())
		return nil
	},
}

// commentsCreateCmd represents the comments create command
var commentsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new comment",
	Long:  `Create a new comment on a fault.` + v3Reference("comments/create-comment"),
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := requireCommentFault(); err != nil {
			return err
		}
		if commentBody == "" {
			return fmt.Errorf("comment body is required. Set it using --body flag")
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		comment, err := client.Faults.AddComment(
			context.Background(),
			commentsProjectID,
			commentsFaultID,
			commentBody,
		)
		if err != nil {
			return dataAPIError("create comment", err)
		}

		if commentsOutputFormat == "json" {
			return printJSON(comment)
		}
		fmt.Printf("Comment created successfully!\n")
		fmt.Printf("  ID: %s\n", comment.Id)
		fmt.Printf("  Body: %s\n", comment.Body.GetOrEmpty())
		return nil
	},
}

// commentsUpdateCmd represents the comments update command
var commentsUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update an existing comment",
	Long:  `Update the body of an existing comment.` + v3Reference("comments/update-comment"),
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := requireCommentFault(); err != nil {
			return err
		}
		if commentID == "" {
			return fmt.Errorf("comment ID is required. Set it using --id flag")
		}
		if commentBody == "" {
			return fmt.Errorf("comment body is required. Set it using --body flag")
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		comment, err := client.Faults.UpdateComment(
			context.Background(),
			commentsProjectID,
			commentsFaultID,
			commentID,
			commentBody,
		)
		if err != nil {
			return dataAPIError("update comment", err)
		}

		if commentsOutputFormat == "json" {
			return printJSON(comment)
		}
		fmt.Printf("Comment updated successfully!\n")
		fmt.Printf("  ID: %s\n", comment.Id)
		fmt.Printf("  Body: %s\n", comment.Body.GetOrEmpty())
		return nil
	},
}

// commentsDeleteCmd represents the comments delete command
var commentsDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a comment",
	Long: `Delete a comment by ID. This action cannot be undone.` + v3Reference(
		"comments/delete-comment",
	),
	RunE: func(_ *cobra.Command, _ []string) error {
		if err := requireCommentFault(); err != nil {
			return err
		}
		if commentID == "" {
			return fmt.Errorf("comment ID is required. Set it using --id flag")
		}

		client, err := newDataAPIClient()
		if err != nil {
			return err
		}

		if err := client.Faults.DeleteComment(
			context.Background(),
			commentsProjectID,
			commentsFaultID,
			commentID,
		); err != nil {
			return dataAPIError("delete comment", err)
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
	commentsCmd.PersistentFlags().StringVar(&commentsProjectID, "project-id", "", "Project ID")
	commentsCmd.PersistentFlags().StringVar(&commentsFaultID, "fault-id", "", "Fault ID")

	// Flags for list command
	commentsListCmd.Flags().
		StringVarP(&commentsOutputFormat, "output", "o", "table", "Output format (table or json)")

	// Flags for get command
	commentsGetCmd.Flags().StringVar(&commentID, "id", "", "Comment ID")
	commentsGetCmd.Flags().
		StringVarP(&commentsOutputFormat, "output", "o", "text", "Output format (text or json)")
	_ = commentsGetCmd.MarkFlagRequired("id")

	// Flags for create command
	commentsCreateCmd.Flags().StringVar(&commentBody, "body", "", "Comment body text")
	commentsCreateCmd.Flags().
		StringVarP(&commentsOutputFormat, "output", "o", "text", "Output format (text or json)")
	_ = commentsCreateCmd.MarkFlagRequired("body")

	// Flags for update command
	commentsUpdateCmd.Flags().StringVar(&commentID, "id", "", "Comment ID")
	commentsUpdateCmd.Flags().StringVar(&commentBody, "body", "", "New comment body text")
	commentsUpdateCmd.Flags().
		StringVarP(&commentsOutputFormat, "output", "o", "text", "Output format (text or json)")
	_ = commentsUpdateCmd.MarkFlagRequired("id")
	_ = commentsUpdateCmd.MarkFlagRequired("body")

	// Flags for delete command
	commentsDeleteCmd.Flags().StringVar(&commentID, "id", "", "Comment ID")
	_ = commentsDeleteCmd.MarkFlagRequired("id")
}
