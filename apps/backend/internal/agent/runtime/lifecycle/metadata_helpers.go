package lifecycle

const StopReasonTaskDeleted = "task deleted"

func boolPtr(v bool) *bool {
	return &v
}

func autoApprovePermissionsOverride(enabled bool, override *bool) *bool {
	if override != nil {
		return override
	}
	if enabled {
		return boolPtr(true)
	}
	return nil
}

func getMetadataString(metadata map[string]interface{}, key string) string {
	if metadata == nil {
		return ""
	}
	if v, ok := metadata[key].(string); ok {
		return v
	}
	return ""
}

func getMetadataStringMap(metadata map[string]interface{}, key string) map[string]string {
	if metadata == nil {
		return nil
	}
	switch v := metadata[key].(type) {
	case map[string]string:
		if len(v) == 0 {
			return nil
		}
		return v
	case map[string]interface{}:
		if len(v) == 0 {
			return nil
		}
		out := make(map[string]string, len(v))
		for k, raw := range v {
			if s, ok := raw.(string); ok {
				out[k] = s
			}
		}
		if len(out) == 0 {
			return nil
		}
		return out
	}
	return nil
}
