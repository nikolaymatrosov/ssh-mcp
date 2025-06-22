package ssh

// SSHConnectArgs defines the arguments for establishing an SSH connection
type SSHConnectArgs struct {
	Host     string `json:"host" jsonschema:"description=The SSH server hostname or IP address,required"`
	Port     int    `json:"port" jsonschema:"description=The SSH server port,default=22"`
	Username string `json:"username" jsonschema:"description=The SSH username,required"`
	Password string `json:"password" jsonschema:"description=The SSH password (leave empty if using key-based auth)"`
	KeyPath  string `json:"keyPath" jsonschema:"description=Path to the private key file (leave empty if using password auth)"`
	Timeout  int    `json:"timeout" jsonschema:"description=Connection timeout in seconds,default=10"`
}

// SSHCommandArgs defines the arguments for executing a command over SSH
type SSHCommandArgs struct {
	SessionID string `json:"sessionId" jsonschema:"description=The SSH session identifier,required"`
	Command   string `json:"command" jsonschema:"description=The command to execute,required"`
	Timeout   int    `json:"timeout" jsonschema:"description=Command execution timeout in seconds,default=30"`
}

// SSHFileTransferArgs defines the arguments for transferring files over SSH
type SSHFileTransferArgs struct {
	SessionID   string `json:"sessionId" jsonschema:"description=The SSH session identifier,required"`
	Source      string `json:"source" jsonschema:"description=Source file path,required"`
	Destination string `json:"destination" jsonschema:"description=Destination file path,required"`
	Direction   string `json:"direction" jsonschema:"description=Transfer direction (upload or download),required,enum=upload,enum=download"`
}

// SSHDirectoryUploadArgs defines the arguments for uploading directories over SSH
type SSHDirectoryUploadArgs struct {
	SessionID   string `json:"sessionId" jsonschema:"description=The SSH session identifier,required"`
	Source      string `json:"source" jsonschema:"description=Source directory path on local machine,required"`
	Destination string `json:"destination" jsonschema:"description=Destination directory path on remote server,required"`
}

// SSHDirectoryDownloadArgs defines the arguments for downloading directories over SSH
type SSHDirectoryDownloadArgs struct {
	SessionID   string `json:"sessionId" jsonschema:"description=The SSH session identifier,required"`
	Source      string `json:"source" jsonschema:"description=Source directory path on remote server,required"`
	Destination string `json:"destination" jsonschema:"description=Destination directory path on local machine,required"`
}

// SSHDisconnectArgs defines the arguments for disconnecting an SSH session
type SSHDisconnectArgs struct {
	SessionID string `json:"sessionId" jsonschema:"description=The SSH session identifier,required"`
}

// SSHListSessionsArgs defines the arguments for listing active SSH sessions
type SSHListSessionsArgs struct {
	// Empty struct as we don't need any arguments for listing sessions
}

// SSHListDirectoryArgs defines the arguments for listing directory contents
type SSHListDirectoryArgs struct {
	SessionID string `json:"sessionId" jsonschema:"description=The SSH session identifier,required"`
	Path      string `json:"path" jsonschema:"description=Directory path to list,required"`
}
