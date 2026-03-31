package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"tieba-cli/internal/api"

	"github.com/spf13/cobra"
)

var token string

var rootCmd = &cobra.Command{
	Use:   "tiecli",
	Short: "Tieba CLI for Baidu Tieba Skill",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if token == "" {
			token = os.Getenv("TB_TOKEN")
		}
		if token == "" {
			fmt.Fprintln(os.Stderr, "Error: TB_TOKEN is required. Set it via --token or TB_TOKEN env var.")
			os.Exit(1)
		}
	},
}

// ── list ─────────────────────────────────────────────────────────────────────

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List threads in the forum",
	Run: func(cmd *cobra.Command, args []string) {
		client := api.NewClient(token)
		res, err := client.ListThreads(0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return
		}
		for _, t := range res.Data.ThreadList {
			fmt.Printf("[%d] %s (by %s, replies: %d)\n", t.ID, t.Title, t.Author.Name, t.ReplyNum)
		}
	},
}

// ── get ──────────────────────────────────────────────────────────────────────
// Output is written to stdout so callers can redirect:
//   tiecli get 123456 > thread.json
//   tiecli get 123456 | jq '.post_list[].content'

var (
	getPage     int
	getAllPages bool
)

var getCmd = &cobra.Command{
	Use:   "get [thread_id]",
	Short: "Print thread JSON to stdout (redirect to save: tiecli get ID > file.json)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var tid int64
		if _, err := fmt.Sscanf(args[0], "%d", &tid); err != nil {
			fmt.Fprintf(os.Stderr, "Invalid thread ID: %s\n", args[0])
			return
		}

		client := api.NewClient(token)

		if !getAllPages {
			// Single page
			res, err := client.GetThreadDetails(tid, getPage)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				return
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			enc.Encode(res)
			return
		}

		// Fetch all pages and stream as a JSON array
		fmt.Print("[\n")
		for pn := 1; ; pn++ {
			res, err := client.GetThreadDetails(tid, pn)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error on page %d: %v\n", pn, err)
				break
			}
			data, _ := json.MarshalIndent(res, "  ", "  ")
			if pn > 1 {
				fmt.Print(",\n")
			}
			fmt.Printf("  %s", data)
			if res.Page.HasMore == 0 || pn >= res.Page.TotalPage {
				break
			}
		}
		fmt.Print("\n]\n")
	},
}

// ── post ─────────────────────────────────────────────────────────────────────

var (
	postTitle   string
	postContent string
	postTabID   int
	postTabName string
)

var postCmd = &cobra.Command{
	Use:   "post",
	Short: "Create a new thread",
	Run: func(cmd *cobra.Command, args []string) {
		client := api.NewClient(token)
		res, err := client.AddThread(postTitle, postContent, postTabID, postTabName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return
		}
		if res.ErrNo != 0 {
			fmt.Fprintf(os.Stderr, "Failed: %s\n", res.ErrMsg)
			return
		}
		fmt.Printf("Thread created: https://tieba.baidu.com/p/%d\n", res.Data.ThreadID)
	},
}

// ── reply ─────────────────────────────────────────────────────────────────────

var (
	replyContent string
	replyPid     int64 // own flag, not shared with agreeCmd
)

var replyCmd = &cobra.Command{
	Use:   "reply [thread_id]",
	Short: "Reply to a thread or a specific post",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var tid int64
		if _, err := fmt.Sscanf(args[0], "%d", &tid); err != nil {
			fmt.Fprintf(os.Stderr, "Invalid thread ID: %s\n", args[0])
			return
		}
		client := api.NewClient(token)
		res, err := client.AddPost(replyContent, tid, replyPid)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return
		}
		if res.ErrNo != 0 {
			fmt.Fprintf(os.Stderr, "Failed: %s\n", res.ErrMsg)
			return
		}
		fmt.Printf("Replied: https://tieba.baidu.com/p/%d?pid=%d\n",
			res.Data.ThreadID, res.Data.PostID)
	},
}

// ── agree ────────────────────────────────────────────────────────────────────

var (
	agreeTid    int64
	agreePid    int64
	agreeType   int
	agreeCancel bool
)

var agreeCmd = &cobra.Command{
	Use:   "agree",
	Short: "Like or unlike a post/thread",
	Run: func(cmd *cobra.Command, args []string) {
		client := api.NewClient(token)
		op := 0
		if agreeCancel {
			op = 1
		}
		res, err := client.OpAgree(agreeTid, agreePid, agreeType, op)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return
		}
		if res.ErrNo == 0 {
			fmt.Println("Success")
		} else {
			fmt.Fprintf(os.Stderr, "Failed: %s\n", res.ErrMsg)
		}
	},
}

// ── inbox ────────────────────────────────────────────────────────────────────

var inboxPage int

var inboxCmd = &cobra.Command{
	Use:   "inbox",
	Short: "Check incoming replies",
	Run: func(cmd *cobra.Command, args []string) {
		client := api.NewClient(token)
		res, err := client.ReplyMe(inboxPage)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return
		}
		for _, r := range res.Data.ReplyList {
			status := " "
			if r.Unread == 1 {
				status = "*"
			}
			fmt.Printf("[%s] thread=%d post=%d %s: %s\n",
				status, r.ThreadID, r.PostID, r.Title, r.Content)
		}
	},
}

// ── profile ───────────────────────────────────────────────────────────────────

var profileName string

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Modify profile (nickname)",
	Run: func(cmd *cobra.Command, args []string) {
		client := api.NewClient(token)
		_, err := client.ModifyName(profileName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return
		}
		fmt.Println("Nickname updated")
	},
}

// ── delete ───────────────────────────────────────────────────────────────────

var deleteCmd = &cobra.Command{
	Use:   "delete [thread|post] [id]",
	Short: "Delete a thread or post",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		client := api.NewClient(token)
		var id int64
		if _, err := fmt.Sscanf(args[1], "%d", &id); err != nil {
			fmt.Fprintf(os.Stderr, "Invalid ID: %s\n", args[1])
			return
		}
		var err error
		if args[0] == "thread" {
			_, err = client.DelThread(id)
		} else if args[0] == "post" {
			_, err = client.DelPost(id)
		} else {
			fmt.Fprintf(os.Stderr, "Unknown type %q: use 'thread' or 'post'\n", args[0])
			return
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		} else {
			fmt.Println("Deleted")
		}
	},
}

// ── subposts ──────────────────────────────────────────────────────────────────

var subpostsCmd = &cobra.Command{
	Use:   "subposts [thread_id] [post_id]",
	Short: "Get replies to a specific post",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		var tid, pid int64
		fmt.Sscanf(args[0], "%d", &tid)
		fmt.Sscanf(args[1], "%d", &pid)
		client := api.NewClient(token)
		res, err := client.GetNestedFloor(tid, pid)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return
		}
		for _, p := range res.Data.PostList {
			txt := ""
			for _, c := range p.Content {
				txt += c.Text
			}
			fmt.Printf("[%d] %s\n", p.ID, txt)
		}
	},
}

// ── init ──────────────────────────────────────────────────────────────────────
// FIX: uses a plain HTTP client (no Authorization header) to download public
// CDN assets, so TB_TOKEN is never sent to a non-tieba.baidu.com domain.

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize skill documentation",
	Run: func(cmd *cobra.Command, args []string) {
		files := map[string]string{
			"SKILL.md":         "https://tieba-ares.cdn.bcebos.com/skill.md",
			"api-reference.md": "https://tieba-ares.cdn.bcebos.com/api-reference.md",
		}
		for name, url := range files {
			fmt.Printf("Downloading %s...\n", name)
			dest, err := api.DownloadPublicFile(url, name)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to download %s: %v\n", name, err)
			} else {
				fmt.Printf("Saved to %s\n", dest)
			}
		}
	},
}

// ── Execute ───────────────────────────────────────────────────────────────────

func Execute() {
	rootCmd.PersistentFlags().StringVarP(&token, "token", "t", "", "Baidu Tieba Token")

	// post flags
	postCmd.Flags().StringVar(&postTitle, "title", "", "Thread title (required)")
	postCmd.Flags().StringVar(&postContent, "content", "", "Thread content (required)")
	postCmd.Flags().IntVar(&postTabID, "tab-id", 0, "Tab ID")
	postCmd.Flags().StringVar(&postTabName, "tab-name", "", "Tab name")
	postCmd.MarkFlagRequired("title")
	postCmd.MarkFlagRequired("content")

	// reply flags — separate from agree variables
	replyCmd.Flags().StringVar(&replyContent, "content", "", "Reply content (required)")
	replyCmd.Flags().Int64Var(&replyPid, "pid", 0, "Post ID to reply to (omit to reply to thread)")
	replyCmd.MarkFlagRequired("content")

	// agree flags
	agreeCmd.Flags().Int64Var(&agreeTid, "tid", 0, "Thread ID (required)")
	agreeCmd.Flags().Int64Var(&agreePid, "pid", 0, "Post ID (omit to agree on thread)")
	agreeCmd.Flags().IntVar(&agreeType, "type", 1, "Obj type: 1=floor 2=sub-floor 3=thread")
	agreeCmd.Flags().BoolVar(&agreeCancel, "cancel", false, "Cancel agree instead of adding")
	agreeCmd.MarkFlagRequired("tid")

	// get flags
	getCmd.Flags().IntVar(&getPage, "page", 1, "Page number (ignored when --all is set)")
	getCmd.Flags().BoolVar(&getAllPages, "all", false, "Fetch all pages and output as a JSON array")

	// inbox flags
	inboxCmd.Flags().IntVar(&inboxPage, "page", 1, "Page number")

	// profile flags
	profileCmd.Flags().StringVar(&profileName, "name", "", "New nickname (required)")
	profileCmd.MarkFlagRequired("name")

	rootCmd.AddCommand(listCmd, getCmd, initCmd, postCmd, replyCmd,
		agreeCmd, inboxCmd, profileCmd, deleteCmd, subpostsCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
