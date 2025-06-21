FROM ubuntu:22.04

# Install OpenSSH Server and other utilities
RUN apt-get update && apt-get install -y \
    openssh-server \
    sudo \
    curl \
    vim \
    && rm -rf /var/lib/apt/lists/*

# Create a test user
RUN useradd -m -d /home/testuser -s /bin/bash testuser && \
    echo "testuser:password" | chpasswd && \
    mkdir -p /home/testuser/.ssh && \
    chmod 700 /home/testuser/.ssh && \
    chown -R testuser:testuser /home/testuser

# Configure SSH server
RUN mkdir -p /var/run/sshd && \
    sed -i 's/#PermitRootLogin prohibit-password/PermitRootLogin no/' /etc/ssh/sshd_config && \
    sed -i 's/#PasswordAuthentication yes/PasswordAuthentication yes/' /etc/ssh/sshd_config && \
    echo "AllowUsers testuser" >> /etc/ssh/sshd_config

# Create test files and directories
RUN mkdir -p /home/testuser/test-dir && \
    echo "This is a test file" > /home/testuser/test-file.txt && \
    chown -R testuser:testuser /home/testuser

# Expose SSH port
EXPOSE 22

# Start SSH server
CMD ["/usr/sbin/sshd", "-D"]