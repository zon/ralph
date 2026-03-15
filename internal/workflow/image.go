package workflow

// Image holds the container image repository and tag for a workflow.
type Image struct {
	Repository string
	Tag        string
}

// MakeImage creates an Image with the given repository and tag.
func MakeImage(repository, tag string) Image {
	return Image{Repository: repository, Tag: tag}
}
