package server

import (
	mcp_golang "github.com/metoro-io/mcp-golang"

	"ssh-mcp/internal/file"
	"ssh-mcp/internal/security"
	"ssh-mcp/internal/session"
	"ssh-mcp/internal/ssh"
)

// Tool represents a tool that can be registered with the MCP server
type Tool struct {
	Name        string
	Description string
	Handler     interface{}
}

// GetTools returns all available tools for the SSH MCP server
func GetTools(sessionManager *session.Manager, securityManager *security.Manager) []Tool {
	sshClient := ssh.NewClient(sessionManager)
	fileOps := file.NewOperations(sessionManager)

	return []Tool{
		{
			Name:        "ssh_connect",
			Description: "Establish an SSH connection",
			Handler: func(args ssh.SSHConnectArgs) (*mcp_golang.ToolResponse, error) {
				// Check security
				if err := securityManager.CheckHost(args.Host); err != nil {
					return mcp_golang.NewToolResponse(mcp_golang.NewTextContent("Security error: " + err.Error())), err
				}

				sessionID, err := sshClient.Connect(args)
				if err != nil {
					return mcp_golang.NewToolResponse(mcp_golang.NewTextContent("Connection error: " + err.Error())), err
				}

				return mcp_golang.NewToolResponse(mcp_golang.NewTextContent("Connected. Session ID: " + sessionID)), nil
			},
		},
		{
			Name:        "ssh_execute",
			Description: "Execute a command over SSH",
			Handler: func(args ssh.SSHCommandArgs) (*mcp_golang.ToolResponse, error) {
				// Check security
				if err := securityManager.CheckCommand(args.SessionID, args.Command); err != nil {
					return mcp_golang.NewToolResponse(mcp_golang.NewTextContent("Security error: " + err.Error())), err
				}

				output, err := sshClient.ExecuteCommand(args)
				if err != nil {
					return mcp_golang.NewToolResponse(mcp_golang.NewTextContent("Command error: " + err.Error())), err
				}

				return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(output)), nil
			},
		},
		{
			Name:        "ssh_disconnect",
			Description: "Close an SSH connection",
			Handler: func(args ssh.SSHDisconnectArgs) (*mcp_golang.ToolResponse, error) {
				err := sshClient.Disconnect(args)
				if err != nil {
					return mcp_golang.NewToolResponse(mcp_golang.NewTextContent("Disconnect error: " + err.Error())), err
				}

				return mcp_golang.NewToolResponse(mcp_golang.NewTextContent("Disconnected session: " + args.SessionID)), nil
			},
		},
		{
			Name:        "ssh_list_sessions",
			Description: "List active SSH sessions",
			Handler: func(args ssh.SSHListSessionsArgs) (*mcp_golang.ToolResponse, error) {
				sessions := sshClient.ListSessions()

				if len(sessions) == 0 {
					return mcp_golang.NewToolResponse(mcp_golang.NewTextContent("No active SSH sessions")), nil
				}

				result := "Active SSH Sessions:\n"
				for _, sess := range sessions {
					result += "- ID: " + sess["id"] + "\n"
					result += "  Host: " + sess["host"] + "\n"
					result += "  Username: " + sess["username"] + "\n"
					result += "  Created: " + sess["createdAt"] + "\n"
					result += "  Last Activity: " + sess["lastActivity"] + "\n\n"
				}

				return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(result)), nil
			},
		},
		{
			Name:        "ssh_upload_file",
			Description: "Upload a file to the SSH server",
			Handler: func(args ssh.SSHFileTransferArgs) (*mcp_golang.ToolResponse, error) {
				if args.Direction != "upload" {
					return mcp_golang.NewToolResponse(mcp_golang.NewTextContent("Direction must be 'upload'")), nil
				}

				err := fileOps.Upload(args.SessionID, args.Source, args.Destination)
				if err != nil {
					return mcp_golang.NewToolResponse(mcp_golang.NewTextContent("Upload error: " + err.Error())), err
				}

				return mcp_golang.NewToolResponse(mcp_golang.NewTextContent("File uploaded successfully")), nil
			},
		},
		{
			Name:        "ssh_download_file",
			Description: "Download a file from the SSH server",
			Handler: func(args ssh.SSHFileTransferArgs) (*mcp_golang.ToolResponse, error) {
				if args.Direction != "download" {
					return mcp_golang.NewToolResponse(mcp_golang.NewTextContent("Direction must be 'download'")), nil
				}

				err := fileOps.Download(args.SessionID, args.Source, args.Destination)
				if err != nil {
					return mcp_golang.NewToolResponse(mcp_golang.NewTextContent("Download error: " + err.Error())), err
				}

				return mcp_golang.NewToolResponse(mcp_golang.NewTextContent("File downloaded successfully")), nil
			},
		},
		{
			Name:        "ssh_list_directory",
			Description: "List contents of a directory on the SSH server",
			Handler: func(args ssh.SSHListDirectoryArgs) (*mcp_golang.ToolResponse, error) {
				files, err := fileOps.ListDirectory(args.SessionID, args.Path)
				if err != nil {
					return mcp_golang.NewToolResponse(mcp_golang.NewTextContent("List directory error: " + err.Error())), err
				}

				if len(files) == 0 {
					return mcp_golang.NewToolResponse(mcp_golang.NewTextContent("Directory is empty")), nil
				}

				result := "Directory contents of " + args.Path + ":\n"
				for _, file := range files {
					isDir := file["isDirectory"] == "true"
					dirMarker := ""
					if isDir {
						dirMarker = "/"
					}

					result += file["permissions"] + " " + file["size"] + " " + file["date"] + " " + file["name"] + dirMarker + "\n"
				}

				return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(result)), nil
			},
		},
		{
			Name:        "ssh_upload_directory",
			Description: "Upload a directory to the SSH server",
			Handler: func(args ssh.SSHDirectoryUploadArgs) (*mcp_golang.ToolResponse, error) {
				err := fileOps.UploadDir(args.SessionID, args.Source, args.Destination)
				if err != nil {
					return mcp_golang.NewToolResponse(mcp_golang.NewTextContent("Directory upload error: " + err.Error())), err
				}

				return mcp_golang.NewToolResponse(mcp_golang.NewTextContent("Directory uploaded successfully")), nil
			},
		},
		{
			Name:        "ssh_download_directory",
			Description: "Download a directory from the SSH server",
			Handler: func(args ssh.SSHDirectoryDownloadArgs) (*mcp_golang.ToolResponse, error) {
				err := fileOps.DownloadDir(args.SessionID, args.Source, args.Destination)
				if err != nil {
					return mcp_golang.NewToolResponse(mcp_golang.NewTextContent("Directory download error: " + err.Error())), err
				}

				return mcp_golang.NewToolResponse(mcp_golang.NewTextContent("Directory downloaded successfully")), nil
			},
		},
	}
}
