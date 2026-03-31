package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"tieba-cli/internal/api"

	"github.com/spf13/cobra"
)

var (
	token string
)

var rootCmd = &cobra.Command{
	Use:   "tiecli",
	Short: "Tieba CLI for Baidu Tieba Skill",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if token == "" {
			token = os.Getenv("TB_TOKEN")
		}
		if token == "" {
			fmt.Println("Error: TB_TOKEN is required. Set it via --token or TB_TOKEN env var.")
			os.Exit(1)
		}
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List threads in the forum",
	Run: func(cmd *cobra.Command, args []string) {
		client := api.NewClient(token)
		res, err := client.ListThreads(0)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		for _, t := range res.Data.ThreadList {
			fmt.Printf("[%d] %s (by %s, replies: %d)\n", t.ID, t.Title, t.Author.Name, t.ReplyNum)
		}
	},
}

var getCmd = &cobra.Command{
	Use:   "get [thread_id]",
	Short: "Download a thread and save it to a file",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var tid int64
		_, err := fmt.Sscanf(args[0], "%d", &tid)
		if err != nil {
			fmt.Printf("Invalid thread ID: %s\n", args[0])
			return
		}

		client := api.NewClient(token)
		res, err := client.GetThreadDetails(tid, 1) // Start with page 1
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		// Save to file
		filename := fmt.Sprintf("thread_%d.json", tid)
		data, _ := json.MarshalIndent(res, "", "  ")
		err = os.WriteFile(filename, data, 0644)
		if err != nil {
			fmt.Printf("Failed to save file: %v\n", err)
			return
		}
		fmt.Printf("Thread %d saved to %s\n", tid, filename)
	},
}

var (
	postTitle   string
	postContent string
	postTabID   int
	postTabName string
	agreeTid    int64
	agreePid    int64
	agreeType   int
	agreeCancel bool
	profileName string
	inboxPage   int
)

var postCmd = &cobra.Command{
	Use:   "post",
	Short: "Create a new thread",
	Run: func(cmd *cobra.Command, args []string) {
		client := api.NewClient(token)
		res, err := client.AddThread(postTitle, postContent, postTabID, postTabName)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Printf("Thread created: https://tieba.baidu.com/p/%d\n", res.Data.ThreadID)
	},
}

var replyCmd = &cobra.Command{
	Use:   "reply [thread_id] [content]",
	Short: "Reply to a thread",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var tid int64
		fmt.Sscanf(args[0], "%d", &tid)
		content := ""
		if len(args) > 1 {
			content = args[1]
		} else {
			content = postContent
		}
		client := api.NewClient(token)
		res, err := client.AddPost(content, tid, agreePid)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		if res.ErrNo != 0 {
			fmt.Printf("Failed: %s\n", res.ErrMsg)
			return
		}
		// Note: AddPost in api-reference.md response for addPost also has data.thread_id
		fmt.Printf("Replied successfully. Check it on Tieba.\n")
	},
}

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
			fmt.Printf("Error: %v\n", err)
			return
		}
		if res.ErrNo == 0 {
			fmt.Println("Success")
		} else {
			fmt.Printf("Failed: %s\n", res.ErrMsg)
		}
	},
}

var inboxCmd = &cobra.Command{
	Use:   "inbox",
	Short: "Check incoming replies",
	Run: func(cmd *cobra.Command, args []string) {
		client := api.NewClient(token)
		res, err := client.ReplyMe(inboxPage)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		for _, r := range res.Data.ReplyList {
			status := " "
			if r.Unread == 1 {
				status = "*"
			}
			fmt.Printf("[%s] [%d] %s: %s\n", status, r.ThreadID, r.Title, r.Content)
		}
	},
}

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Modify profile (nickname)",
	Run: func(cmd *cobra.Command, args []string) {
		client := api.NewClient(token)
		_, err := client.ModifyName(profileName)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Println("Nickname updated")
	},
}

var deleteCmd = &cobra.Command{
	Use:   "delete [type] [id]",
	Short: "Delete a thread or post",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		client := api.NewClient(token)
		id := int64(0)
		fmt.Sscanf(args[1], "%d", &id)
		var err error
		if args[0] == "thread" {
			_, err = client.DelThread(id)
		} else {
			_, err = client.DelPost(id)
		}
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			fmt.Println("Deleted")
		}
	},
}

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
			fmt.Printf("Error: %v\n", err)
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

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize skill documentation",
	Run: func(cmd *cobra.Command, args []string) {
		dir := filepath.Join(os.Getenv("HOME"), ".openclaw", "skills", "tieba-claw")
		os.MkdirAll(dir, 0755)

		files := map[string]string{
			"SKILL.md":         "https://tieba-ares.cdn.bcebos.com/skill.md",
			"api-reference.md": "https://tieba-ares.cdn.bcebos.com/api-reference.md",
		}

		client := api.NewClient(token)
		for name, url := range files {
			fmt.Printf("Downloading %s...\n", name)
			path := filepath.Join(dir, name)
			_, err := client.DownloadFile(url, path)
			if err != nil {
				fmt.Printf("Failed to download %s: %v\n", name, err)
			} else {
				fmt.Printf("Saved to %s\n", path)
			}
		}
	},
}

func Execute() {
	rootCmd.PersistentFlags().StringVarP(&token, "token", "t", "", "Baidu Tieba Token")

	postCmd.Flags().StringVar(&postTitle, "title", "", "Thread title")
	postCmd.Flags().StringVar(&postContent, "content", "", "Thread content")
	postCmd.Flags().IntVar(&postTabID, "tab-id", 0, "Tab ID")
	postCmd.Flags().StringVar(&postTabName, "tab-name", "", "Tab Name")

	agreeCmd.Flags().Int64Var(&agreeTid, "tid", 0, "Thread ID")
	agreeCmd.Flags().Int64Var(&agreePid, "pid", 0, "Post ID")
	agreeCmd.Flags().IntVar(&agreeType, "type", 1, "Obj Type (1:Floor, 2:LCL, 3:Thread)")
	agreeCmd.Flags().BoolVar(&agreeCancel, "cancel", false, "Cancel agree")

	inboxCmd.Flags().IntVar(&inboxPage, "page", 1, "Page number")
	profileCmd.Flags().StringVar(&profileName, "name", "", "New nickname")

	rootCmd.AddCommand(listCmd, getCmd, initCmd, postCmd, replyCmd, agreeCmd, inboxCmd, profileCmd, deleteCmd, subpostsCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
