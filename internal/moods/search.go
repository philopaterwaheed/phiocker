package moods

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

func Search(query string, limit int) {
	fmt.Printf("Searching for images: %s\n\n", query)

	if strings.Contains(query, ":") {
		// Search for specific image
		fmt.Printf("Getting detailed information for image: %s\n", query)
		if err := SearchImageInfo(query); err != nil {
			fmt.Printf("Error searching for image info '%s': %v\n", query, err)
			return
		}
	} else {
		fmt.Printf("Searching for available tags in repository: %s\n", query)
		if err := SearchImageTags(query, limit); err != nil {
			fmt.Printf("Error searching for tags in repository '%s': %v\n", query, err)
			fmt.Println("\nTrying to get information for image with 'latest' tag...")
			latestQuery := query + ":latest"
			if err := SearchImageInfo(latestQuery); err != nil {
				fmt.Printf("Error getting image info for '%s': %v\n", latestQuery, err)
				fmt.Println("\nTips:")
				fmt.Println("  - Make sure the repository exists and is publicly accessible")
				fmt.Println("  - Try searching with a specific tag (e.g., ubuntu:20.04)")
				fmt.Println("  - Check if you need authentication for private repositories")
			}
		}
	}
}

func SearchImageTags(repository string, limit int) error {
	fmt.Printf("Searching for available tags in repository: %s\n", repository)

	repo, err := name.NewRepository(repository)
	if err != nil {
		return fmt.Errorf("failed to parse repository %s: %v", repository, err)
	}

	tags, err := remote.List(repo, remote.WithAuthFromKeychain(authn.DefaultKeychain), remote.WithContext(context.Background()))
	if err != nil {
		return fmt.Errorf("failed to list tags for repository %s: %v", repository, err)
	}

	if len(tags) == 0 {
		fmt.Printf("No tags found for repository '%s'\n", repository)
		return nil
	}

	sort.Strings(tags)

	fmt.Printf("Found %d tag(s) for repository '%s':\n", len(tags), repository)

	maxDisplay := limit
	if maxDisplay <= 0 {
		maxDisplay = 50
	}

	for i, tag := range tags {
		if i < maxDisplay {
			fmt.Printf("  %s\n", tag)
		} else if i == maxDisplay && len(tags) > maxDisplay {
			fmt.Printf("  ... and %d more tags\n", len(tags)-maxDisplay)
			break
		}
	}

	fmt.Println("\nTo download any of these images, use:")
	fmt.Printf("  phiocker download %s:<tag>\n", repository)
	fmt.Println("\nFor detailed information about a specific tag, use:")
	fmt.Printf("  phiocker search %s:<tag>\n", repository)

	return nil
}

func SearchImageInfo(imageRef string) error {
	fmt.Printf("Getting information for image: %s\n", imageRef)

	ref, err := name.ParseReference(imageRef)
	if err != nil {
		return fmt.Errorf("failed to parse image reference %s: %v", imageRef, err)
	}

	manifest, err := remote.Get(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain), remote.WithContext(context.Background()))
	if err != nil {
		return fmt.Errorf("failed to get manifest for %s: %v", imageRef, err)
	}

	img, err := remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err == nil {
		config, err := img.ConfigFile()
		if err == nil {
			fmt.Printf("\nImage Details:\n")
			fmt.Printf("  Repository: %s\n", ref.Context().String())
			fmt.Printf("  Tag: %s\n", ref.Identifier())
			fmt.Printf("  Digest: %s\n", manifest.Digest.String())
			fmt.Printf("  Size: %d bytes\n", manifest.Size)
			fmt.Printf("  Architecture: %s\n", config.Architecture)
			fmt.Printf("  OS: %s\n", config.OS)

			if config.Config.WorkingDir != "" {
				fmt.Printf("  Working Directory: %s\n", config.Config.WorkingDir)
			}

			if len(config.Config.Env) > 0 {
				fmt.Printf("  Environment Variables:\n")
				for _, env := range config.Config.Env {
					fmt.Printf("    %s\n", env)
				}
			}

			if len(config.Config.Cmd) > 0 {
				fmt.Printf("  Default Command: %s\n", strings.Join(config.Config.Cmd, " "))
			}
		}
	}

	fmt.Printf("\nTo download this image, use:\n")
	fmt.Printf("  phiocker download %s\n", imageRef)

	return nil
}
