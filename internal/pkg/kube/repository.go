package kube

type repository struct {
	client *Client
}

func NewKubeRepository(client *Client) Repository {
	return &repository{
		client: client,
	}
}

func (r repository) GetContexts() ([]string, error) {
	var contexts []string

	for context := range r.client.CmdConfig.Contexts {
		contexts = append(contexts, context)
	}

	return contexts, nil
}
