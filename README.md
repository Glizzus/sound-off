# Sound-Off

Sound-Off is the punctual and annoying Disord bot. You specify an audio file and a cron pattern,
and it will join the most populated voice channel in your server and play the audio file at the specified times.

Example scenarios:

- Ring a bell at the top of every hour

- Play _1st of the Month_ by Bone Thugs-N-Harmony on the first of every month

- "It's wednesday, my dudes!" every Wednesday at 8:00 AM

## Development

Sound-Off is primarily developed in VS Code Dev Containers. This allows for a consistent development environment across different machines.

To get started, clone the repository and open it in VS Code. If you have the Remote - Containers extension installed, it will prompt you to reopen the folder in a container.

For more detailed project-specific information, refer to the [Dev Container README](.devcontainer/README.md).

## Stack

Sound-Off is built with a backend architecture that emphasizes reliability and scalability. 

### Backend

- Golang - chosen for performance and concurrency

- Redis - used for job scheduling and state management

- PostgreSQL - used for persistent storage of scheduled jobs

### Object Storage

- MinIO - S3-compatible object storage for storing audio files

### Orchestration

- Docker - used to build images and run infrastructure containers

- Kubernetes - used for deploying and managing the application in a cloud-native environment

### Infrastructure-as-Code

- Terraform - used for provisioning and managing cloud resources

- Ansible - used for configuration management and deployment automation
