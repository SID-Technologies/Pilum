package docker

func GenerateDockerPushCommand(imageName string) []string {
	return []string{"docker", "push", imageName}
}
