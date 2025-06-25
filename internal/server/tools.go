package server

import (
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"

	"ssh-mcp/internal/file"
	"ssh-mcp/internal/security"
	"ssh-mcp/internal/session"
	"ssh-mcp/internal/ssh"
)

// getStringOrEmpty safely converts an interface value to string
// Returns empty string if the value is nil
func getStringOrEmpty(value interface{}) string {
	if value == nil {
		return ""
	}
	if str, ok := value.(string); ok {
		return str
	}
	return ""
}

// getIntOrDefault safely converts an interface value to int
// Returns defaultValue if the value is nil or cannot be converted to int
func getIntOrDefault(value interface{}, defaultValue int) int {
	if value == nil {
		return defaultValue
	}

	switch v := value.(type) {
	case int:
		return v
	case float64:
		return int(v)
	case string:
		// Try to parse string to int, but return default on failure
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}

	return defaultValue
}

// Tool represents a tool that can be registered with the MCP server
type Tool struct {
	Name    string
	Opts    []mcp.ToolOption
	Handler interface{}
}

// GetTools returns all available tools for the SSH MCP server
func GetTools(sessionManager *session.Manager, securityManager *security.Manager) []Tool {
	sshClient := ssh.NewClient(sessionManager)
	fileOps := file.NewOperations(sessionManager)

	return []Tool{
		{
			Name: "ssh_connect",
			Opts: []mcp.ToolOption{
				mcp.WithDescription("Establish an SSH connection"),
				mcp.WithString("host",
					mcp.Required(),
					mcp.Description("The hostname or IP address of the SSH server"),
				),
				mcp.WithNumber("port",
					mcp.DefaultNumber(22),
					mcp.Description("The port number of the SSH server"),
				),
				mcp.WithString("username",
					mcp.Required(),
					mcp.Description("The username to authenticate with"),
				),
				mcp.WithString("password",
					mcp.DefaultString(""),
					mcp.Description("Password for authentication. If using a private key, this can be left empty."),
				),
				mcp.WithString("keyPath",
					mcp.DefaultString(""),
					mcp.Description("Path to the private key file for authentication. If using password, this can be left empty."),
				),
			},
			Handler: func(args map[string]interface{}) (*mcp.CallToolResult, error) {
				// Convert map to SSHConnectArgs
				connectArgs := ssh.SSHConnectArgs{
					Host:     getStringOrEmpty(args["host"]),
					Port:     getIntOrDefault(args["port"], 22),
					Username: getStringOrEmpty(args["username"]),
					Password: getStringOrEmpty(args["password"]),
					KeyPath:  getStringOrEmpty(args["keyPath"]),
				}

				// Check security
				if err := securityManager.CheckHost(connectArgs.Host); err != nil {
					return &mcp.CallToolResult{
						Content: []mcp.Content{
							mcp.TextContent{
								Type: "text",
								Text: "Security error: " + err.Error(),
							},
						},
					}, err
				}

				sessionID, err := sshClient.Connect(connectArgs)
				if err != nil {
					return &mcp.CallToolResult{
						Content: []mcp.Content{
							mcp.TextContent{
								Type: "text",
								Text: "Connection error: " + err.Error(),
							},
						},
					}, err
				}

				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.TextContent{
							Type: "text",
							Text: "Connected. Session ID: " + sessionID,
						},
					},
				}, nil
			},
		},
		{
			Name: "ssh_execute",
			Opts: []mcp.ToolOption{
				mcp.WithDescription("Execute a command over SSH"),
				mcp.WithString("sessionId",
					mcp.Required(),
					mcp.Description("The SSH session identifier"),
				),
				mcp.WithString("command",
					mcp.Required(),
					mcp.Description("The command to execute"),
				),
				mcp.WithNumber("timeout",
					mcp.DefaultNumber(30),
					mcp.Description("Command execution timeout in seconds"),
				),
			},
			Handler: func(args map[string]interface{}) (*mcp.CallToolResult, error) {
				// Convert map to SSHCommandArgs
				commandArgs := ssh.SSHCommandArgs{
					SessionID: getStringOrEmpty(args["sessionId"]),
					Command:   getStringOrEmpty(args["command"]),
				}

				// Check security
				if err := securityManager.CheckCommand(commandArgs.SessionID, commandArgs.Command); err != nil {
					return &mcp.CallToolResult{
						Content: []mcp.Content{
							mcp.TextContent{
								Type: "text",
								Text: "Security error: " + err.Error(),
							},
						},
					}, err
				}

				output, err := sshClient.ExecuteCommand(commandArgs)
				if err != nil {
					return &mcp.CallToolResult{
						Content: []mcp.Content{
							mcp.TextContent{
								Type: "text",
								Text: "Command error: " + err.Error(),
							},
						},
					}, err
				}

				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.TextContent{
							Type: "text",
							Text: output,
						},
					},
				}, nil
			},
		},
		{
			Name: "ssh_disconnect",
			Opts: []mcp.ToolOption{
				mcp.WithDescription("Close an SSH connection"),
				mcp.WithString("sessionId",
					mcp.Required(),
					mcp.Description("The SSH session identifier"),
				),
			},
			Handler: func(args map[string]interface{}) (*mcp.CallToolResult, error) {
				// Convert map to SSHDisconnectArgs
				disconnectArgs := ssh.SSHDisconnectArgs{
					SessionID: getStringOrEmpty(args["sessionId"]),
				}

				err := sshClient.Disconnect(disconnectArgs)
				if err != nil {
					return &mcp.CallToolResult{
						Content: []mcp.Content{
							mcp.TextContent{
								Type: "text",
								Text: "Disconnect error: " + err.Error(),
							},
						},
					}, err
				}

				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.TextContent{
							Type: "text",
							Text: "Disconnected session: " + disconnectArgs.SessionID,
						},
					},
				}, nil
			},
		},
		{
			Name: "ssh_list_sessions",
			Opts: []mcp.ToolOption{
				mcp.WithDescription("List active SSH sessions"),
			},
			Handler: func(args map[string]interface{}) (*mcp.CallToolResult, error) {
				sessions := sshClient.ListSessions()

				if len(sessions) == 0 {
					return &mcp.CallToolResult{
						Content: []mcp.Content{
							mcp.TextContent{
								Type: "text",
								Text: "No active SSH sessions",
							},
						},
					}, nil
				}

				result := "Active SSH Sessions:\n"
				for _, sess := range sessions {
					result += "- ID: " + sess["id"] + "\n"
					result += "  Host: " + sess["host"] + "\n"
					result += "  Username: " + sess["username"] + "\n"
					result += "  Created: " + sess["createdAt"] + "\n"
					result += "  Last Activity: " + sess["lastActivity"] + "\n\n"
				}

				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.TextContent{
							Type: "text",
							Text: result,
						},
					},
				}, nil
			},
		},
		{
			Name: "ssh_upload_file",
			Opts: []mcp.ToolOption{
				mcp.WithDescription("Upload a file to the SSH server"),
				mcp.WithString("sessionId",
					mcp.Required(),
					mcp.Description("The SSH session identifier"),
				),
				mcp.WithString("source",
					mcp.Required(),
					mcp.Description("Source file path"),
				),
				mcp.WithString("destination",
					mcp.Required(),
					mcp.Description("Destination file path"),
				),
			},
			Handler: func(args map[string]interface{}) (*mcp.CallToolResult, error) {
				// Convert map to SSHFileTransferArgs
				transferArgs := ssh.SSHFileTransferArgs{
					SessionID:   getStringOrEmpty(args["sessionId"]),
					Source:      getStringOrEmpty(args["source"]),
					Destination: getStringOrEmpty(args["destination"]),
					Direction:   "upload",
				}

				err := fileOps.Upload(transferArgs.SessionID, transferArgs.Source, transferArgs.Destination)
				if err != nil {
					return &mcp.CallToolResult{
						Content: []mcp.Content{
							mcp.TextContent{
								Type: "text",
								Text: "Upload error: " + err.Error(),
							},
						},
					}, err
				}

				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.TextContent{
							Type: "text",
							Text: "File uploaded successfully",
						},
					},
				}, nil
			},
		},
		{
			Name: "ssh_download_file",
			Opts: []mcp.ToolOption{
				mcp.WithDescription("Download a file from the SSH server"),
				mcp.WithString("sessionId",
					mcp.Required(),
					mcp.Description("The SSH session identifier"),
				),
				mcp.WithString("source",
					mcp.Required(),
					mcp.Description("Source file path"),
				),
				mcp.WithString("destination",
					mcp.Required(),
					mcp.Description("Destination file path"),
				),
			},
			Handler: func(args map[string]interface{}) (*mcp.CallToolResult, error) {
				// Convert map to SSHFileTransferArgs
				transferArgs := ssh.SSHFileTransferArgs{
					SessionID:   getStringOrEmpty(args["sessionId"]),
					Source:      getStringOrEmpty(args["source"]),
					Destination: getStringOrEmpty(args["destination"]),
					Direction:   "download",
				}

				err := fileOps.Download(transferArgs.SessionID, transferArgs.Source, transferArgs.Destination)
				if err != nil {
					return &mcp.CallToolResult{
						Content: []mcp.Content{
							mcp.TextContent{
								Type: "text",
								Text: "Download error: " + err.Error(),
							},
						},
					}, err
				}

				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.TextContent{
							Type: "text",
							Text: "File downloaded successfully",
						},
					},
				}, nil
			},
		},
		{
			Name: "ssh_list_directory",
			Opts: []mcp.ToolOption{
				mcp.WithDescription("List contents of a directory on the SSH server"),
				mcp.WithString("sessionId",
					mcp.Required(),
					mcp.Description("The SSH session identifier"),
				),
				mcp.WithString("path",
					mcp.Required(),
					mcp.Description("Directory path to list"),
				),
			},
			Handler: func(args map[string]interface{}) (*mcp.CallToolResult, error) {
				// Convert map to SSHListDirectoryArgs
				listArgs := ssh.SSHListDirectoryArgs{
					SessionID: getStringOrEmpty(args["sessionId"]),
					Path:      getStringOrEmpty(args["path"]),
				}

				files, err := fileOps.ListDirectory(listArgs.SessionID, listArgs.Path)
				if err != nil {
					return &mcp.CallToolResult{
						Content: []mcp.Content{
							mcp.TextContent{
								Type: "text",
								Text: "List directory error: " + err.Error(),
							},
						},
					}, err
				}

				if len(files) == 0 {
					return &mcp.CallToolResult{
						Content: []mcp.Content{
							mcp.TextContent{
								Type: "text",
								Text: "Directory is empty",
							},
						},
					}, nil
				}

				result := "Directory contents of " + listArgs.Path + ":\n"
				for _, file := range files {
					isDir := file["isDirectory"] == "true"
					dirMarker := ""
					if isDir {
						dirMarker = "/"
					}

					result += file["permissions"] + " " + file["size"] + " " + file["date"] + " " + file["name"] + dirMarker + "\n"
				}

				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.TextContent{
							Type: "text",
							Text: result,
						},
					},
				}, nil
			},
		},
		{
			Name: "ssh_upload_directory",
			Opts: []mcp.ToolOption{
				mcp.WithDescription("Upload a directory to the SSH server"),
				mcp.WithString("sessionId",
					mcp.Required(),
					mcp.Description("The SSH session identifier"),
				),
				mcp.WithString("source",
					mcp.Required(),
					mcp.Description("Source directory path on local machine"),
				),
				mcp.WithString("destination",
					mcp.Required(),
					mcp.Description("Destination directory path on remote server"),
				),
			},
			Handler: func(args map[string]interface{}) (*mcp.CallToolResult, error) {
				// Convert map to SSHDirectoryUploadArgs
				uploadArgs := ssh.SSHDirectoryUploadArgs{
					SessionID:   getStringOrEmpty(args["sessionId"]),
					Source:      getStringOrEmpty(args["source"]),
					Destination: getStringOrEmpty(args["destination"]),
				}

				err := fileOps.UploadDir(uploadArgs.SessionID, uploadArgs.Source, uploadArgs.Destination)
				if err != nil {
					return &mcp.CallToolResult{
						Content: []mcp.Content{
							mcp.TextContent{
								Type: "text",
								Text: "Directory upload error: " + err.Error(),
							},
						},
					}, err
				}

				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.TextContent{
							Type: "text",
							Text: "Directory uploaded successfully",
						},
					},
				}, nil
			},
		},
		{
			Name: "ssh_download_directory",
			Opts: []mcp.ToolOption{
				mcp.WithDescription("Download a directory from the SSH server"),
				mcp.WithString("sessionId",
					mcp.Required(),
					mcp.Description("The SSH session identifier"),
				),
				mcp.WithString("source",
					mcp.Required(),
					mcp.Description("Source directory path on remote server"),
				),
				mcp.WithString("destination",
					mcp.Required(),
					mcp.Description("Destination directory path on local machine"),
				),
			},
			Handler: func(args map[string]interface{}) (*mcp.CallToolResult, error) {
				// Convert map to SSHDirectoryDownloadArgs
				downloadArgs := ssh.SSHDirectoryDownloadArgs{
					SessionID:   getStringOrEmpty(args["sessionId"]),
					Source:      getStringOrEmpty(args["source"]),
					Destination: getStringOrEmpty(args["destination"]),
				}

				err := fileOps.DownloadDir(downloadArgs.SessionID, downloadArgs.Source, downloadArgs.Destination)
				if err != nil {
					return &mcp.CallToolResult{
						Content: []mcp.Content{
							mcp.TextContent{
								Type: "text",
								Text: "Directory download error: " + err.Error(),
							},
						},
					}, err
				}

				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.TextContent{
							Type: "text",
							Text: "Directory downloaded successfully",
						},
					},
				}, nil
			},
		},
	}
}
