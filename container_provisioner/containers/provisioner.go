package containers

import (
	"container_provisioner/utils"
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

// Scrape creates a container, runs it, tails the log and wait for it to exit, and export the file name
func Scrape(fileSuffix string, uploadIdentifier string, hotelName string, containerId string) {

	// Dedicated context and client for each function call
	ctx1 := context.Background()
	cli1, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	utils.ErrorHandler(err)
	defer cli1.Close()

	// Start the container
	err = cli1.ContainerStart(ctx1, containerId, types.ContainerStartOptions{})
	utils.ErrorHandler(err)

	// Wait for the container to exit
	statusCh, errCh := cli1.ContainerWait(ctx1, containerId, container.WaitConditionNotRunning)

	// ContainerWait returns 2 channels. One for the status and one for the wait error (not execution error)
	select {
	case err := <-errCh:
		utils.ErrorHandler(err)

	case status := <-statusCh:
		// If the container exited with non-zero status code, remove the container and return an error
		if status.StatusCode != 0 {
			removeContainer(containerId)
			return
		}
	}

	// The file path in the container
	filePathInContainer := "/puppeteer/reviews/All.csv"

	// Read the file from the container as a reader interface of a tar stream
	fileReader, _, err := cli1.CopyFromContainer(ctx1, containerId, filePathInContainer)
	utils.ErrorHandler(err)

	// Write the file to the host
	exportedFileName := utils.WriteToFileFromTarStream(hotelName, fileSuffix, fileReader)

	// Read the exported csv file
	file := utils.ReadFromFile(exportedFileName)

	// Upload the file to R2
	utils.R2UploadObject(exportedFileName, uploadIdentifier, file)

	// Remove the container
	removeContainer(containerId)
}
