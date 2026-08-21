// Package vault resolves secrets through HashiCorp Vault.
package vault

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/api/cliconfig"
)

// Resolver reads string values from Vault's logical API. Vault client settings,
// including VAULT_ADDR, VAULT_TOKEN, VAULT_NAMESPACE, and TLS options, are read
// from the standard environment variables supported by the official client.
type Resolver struct {
	newClient func() (*vaultapi.Client, error)
}

// NewResolver creates a resolver backed by HashiCorp's official Vault client.
func NewResolver() *Resolver {
	return &Resolver{newClient: newVaultClient}
}

func newVaultClient() (*vaultapi.Client, error) {
	client, err := vaultapi.NewClient(nil)
	if err != nil {
		return nil, err
	}
	if client.Token() != "" {
		return client, nil
	}

	helper, err := cliconfig.DefaultTokenHelper()
	if err != nil {
		return nil, fmt.Errorf("load Vault token helper: %w", err)
	}
	token, err := helper.Get()
	if err != nil {
		return nil, fmt.Errorf("read Vault token helper: %w", err)
	}
	if token != "" {
		client.SetToken(token)
	}
	return client, nil
}

// Resolve reads selectors from the secret at path. Selectors use dot notation
// for nested maps. KV v2 responses are unwrapped automatically; callers should
// provide the full logical path, including data/.
func (r *Resolver) Resolve(ctx context.Context, path string, selectors ...string) (map[string]string, error) {
	client, err := r.newClient()
	if err != nil {
		return nil, fmt.Errorf("create Vault client: %w", err)
	}

	secret, err := client.Logical().ReadWithContext(ctx, strings.TrimPrefix(path, "/"))
	if err != nil {
		return nil, fmt.Errorf("read Vault secret %q: %w", path, err)
	}
	if secret == nil {
		return nil, fmt.Errorf("Vault secret %q was not found", path)
	}

	data := secret.Data
	if nested, ok := kvV2Data(data); ok {
		data = nested
	}

	values := make(map[string]string, len(selectors))
	for _, selector := range selectors {
		value, ok := lookup(data, selector)
		if !ok {
			return nil, fmt.Errorf("Vault secret %q does not contain key %q", path, selector)
		}
		text, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("Vault secret %q key %q is not a string", path, selector)
		}
		values[selector] = text
	}
	return values, nil
}

func kvV2Data(data map[string]any) (map[string]any, bool) {
	nested, dataOK := data["data"].(map[string]any)
	metadata, metadataOK := data["metadata"].(map[string]any)
	return nested, dataOK && metadataOK && isKVV2Metadata(metadata)
}

func isKVV2Metadata(metadata map[string]any) bool {
	versionNumber, versionOK := metadata["version"].(json.Number)
	version, versionErr := versionNumber.Int64()
	created, createdOK := metadata["created_time"].(string)
	deletion, deletionOK := metadata["deletion_time"].(string)
	_, destroyedOK := metadata["destroyed"].(bool)
	return versionOK && versionErr == nil && version > 0 &&
		createdOK && validVaultTimestamp(created) && deletionOK &&
		(deletion == "" || validVaultTimestamp(deletion)) && destroyedOK
}

func validVaultTimestamp(value string) bool {
	_, err := time.Parse(time.RFC3339Nano, value)
	return err == nil
}

func lookup(data map[string]any, selector string) (any, bool) {
	var current any = data
	for _, segment := range strings.Split(selector, ".") {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = object[segment]
		if !ok {
			return nil, false
		}
	}
	return current, true
}
