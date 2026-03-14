package resend_test

import (
	"context"
	"fmt"
	"log"

	"github.com/testcontainers/testcontainers-go"

	"github.com/mdelapenya/testcontainers-go-resend"
)

func ExampleRun() {
	// runContainer {
	ctx := context.Background()

	ctr, err := resend.Run(ctx, resend.DefaultImage)
	defer func() {
		if err := testcontainers.TerminateContainer(ctr); err != nil {
			log.Printf("failed to terminate container: %s", err)
		}
	}()
	if err != nil {
		log.Printf("failed to start container: %s", err)
		return
	}
	// }

	state, err := ctr.State(ctx)
	if err != nil {
		log.Printf("failed to get container state: %s", err)
		return
	}

	fmt.Println(state.Running)

	// Output:
	// true
}

func ExampleContainer_BaseURL() {
	ctx := context.Background()

	ctr, err := resend.Run(ctx, resend.DefaultImage)
	defer func() {
		if err := testcontainers.TerminateContainer(ctr); err != nil {
			log.Printf("failed to terminate container: %s", err)
		}
	}()
	if err != nil {
		log.Printf("failed to start container: %s", err)
		return
	}

	// baseURL {
	baseURL, err := ctr.BaseURL(ctx)
	// }
	if err != nil {
		log.Printf("failed to get base URL: %s", err)
		return
	}

	fmt.Println(baseURL != "")

	// Output:
	// true
}
