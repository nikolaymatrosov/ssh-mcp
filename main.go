package main

import (
	"log"
	"time"

	mcp_golang "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/http"

	"ssh-mcp/internal/file"
	"ssh-mcp/internal/security"
	"ssh-mcp/internal/session"
	"ssh-mcp/internal/ssh"
)

func main() {
	// Create an HTTP transport that listens on /mcp endpoint
	transport := http.NewHTTPTransport("/mcp").WithAddr(":8081")

	// Create a new server with the transport
	server := mcp_golang.NewServer(
		transport,
		mcp_golang.WithName("ssh-mcp"),
		mcp_golang.WithInstructions("SSH client with MCP interface"),
		mcp_golang.WithVersion("0.1.0"),
	)

	// Initialize components
	sessionManager := session.NewManager(30 * time.Minute)
	sessionManager.StartCleanupRoutine(5 * time.Minute)

	securityManager := security.NewManager(security.Config{
		LoggingEnabled: true,
		RateLimit:      time.Second * 1, // 1 operation per second
	})
	securityManager.StartCleanupRoutine(5 * time.Minute, 30 * time.Minute)

	sshClient := ssh.NewClient(sessionManager)
	fileOps := file.NewOperations(sessionManager)

	// Register SSH Connect tool
	err := server.RegisterTool("ssh_connect", "Establish an SSH connection", 
		func(args ssh.SSHConnectArgs) (*mcp_golang.ToolResponse, error) {
			// Check security
			if err := securityManager.CheckHost(args.Host); err != nil {
				return mcp_golang.NewToolResponse(mcp_golang.NewTextContent("Security error: " + err.Error())), err
			}

			sessionID, err := sshClient.Connect(args)
			if err != nil {
				return mcp_golang.NewToolResponse(mcp_golang.NewTextContent("Connection error: " + err.Error())), err
			}

			return mcp_golang.NewToolResponse(mcp_golang.NewTextContent("Connected. Session ID: " + sessionID)), nil
		})
	if err != nil {
		panic(err)
	}

	// Register SSH Execute Command tool
	err = server.RegisterTool("ssh_execute", "Execute a command over SSH", 
		func(args ssh.SSHCommandArgs) (*mcp_golang.ToolResponse, error) {
			// Check security
			if err := securityManager.CheckCommand(args.SessionID, args.Command); err != nil {
				return mcp_golang.NewToolResponse(mcp_golang.NewTextContent("Security error: " + err.Error())), err
			}

			output, err := sshClient.ExecuteCommand(args)
			if err != nil {
				return mcp_golang.NewToolResponse(mcp_golang.NewTextContent("Command error: " + err.Error())), err
			}

			return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(output)), nil
		})
	if err != nil {
		panic(err)
	}

	// Register SSH Disconnect tool
	err = server.RegisterTool("ssh_disconnect", "Close an SSH connection", 
		func(args ssh.SSHDisconnectArgs) (*mcp_golang.ToolResponse, error) {
			err := sshClient.Disconnect(args)
			if err != nil {
				return mcp_golang.NewToolResponse(mcp_golang.NewTextContent("Disconnect error: " + err.Error())), err
			}

			return mcp_golang.NewToolResponse(mcp_golang.NewTextContent("Disconnected session: " + args.SessionID)), nil
		})
	if err != nil {
		panic(err)
	}

	// Register SSH List Sessions tool
	err = server.RegisterTool("ssh_list_sessions", "List active SSH sessions", 
		func(args ssh.SSHListSessionsArgs) (*mcp_golang.ToolResponse, error) {
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
		})
	if err != nil {
		panic(err)
	}

	// Register File Upload tool
	err = server.RegisterTool("ssh_upload_file", "Upload a file to the SSH server", 
		func(args ssh.SSHFileTransferArgs) (*mcp_golang.ToolResponse, error) {
			if args.Direction != "upload" {
				return mcp_golang.NewToolResponse(mcp_golang.NewTextContent("Direction must be 'upload'")), nil
			}

			err := fileOps.Upload(args.SessionID, args.Source, args.Destination)
			if err != nil {
				return mcp_golang.NewToolResponse(mcp_golang.NewTextContent("Upload error: " + err.Error())), err
			}

			return mcp_golang.NewToolResponse(mcp_golang.NewTextContent("File uploaded successfully")), nil
		})
	if err != nil {
		panic(err)
	}

	// Register File Download tool
	err = server.RegisterTool("ssh_download_file", "Download a file from the SSH server", 
		func(args ssh.SSHFileTransferArgs) (*mcp_golang.ToolResponse, error) {
			if args.Direction != "download" {
				return mcp_golang.NewToolResponse(mcp_golang.NewTextContent("Direction must be 'download'")), nil
			}

			err := fileOps.Download(args.SessionID, args.Source, args.Destination)
			if err != nil {
				return mcp_golang.NewToolResponse(mcp_golang.NewTextContent("Download error: " + err.Error())), err
			}

			return mcp_golang.NewToolResponse(mcp_golang.NewTextContent("File downloaded successfully")), nil
		})
	if err != nil {
		panic(err)
	}

	// Register List Directory tool
	err = server.RegisterTool("ssh_list_directory", "List contents of a directory on the SSH server", 
		func(args ssh.SSHListDirectoryArgs) (*mcp_golang.ToolResponse, error) {
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
		})
	if err != nil {
		panic(err)
	}

	// Start the server
	log.Println("Starting SSH-MCP server on :8081...")
	server.Serve()
}
