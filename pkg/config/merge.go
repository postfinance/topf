package config

// deepMerge merges source map into destination map recursively
// Values from source override values in destination
// Returns the merged map (modifies destination in place)
func deepMerge(dst, src map[string]any) map[string]any {
	if dst == nil {
		dst = make(map[string]any)
	}

	for key, srcVal := range src {
		if dstVal, exists := dst[key]; exists {
			// Both exist - check if both are maps for recursive merge
			srcMap, srcIsMap := srcVal.(map[string]any)
			dstMap, dstIsMap := dstVal.(map[string]any)

			if srcIsMap && dstIsMap {
				// Recursively merge nested maps
				dst[key] = deepMerge(dstMap, srcMap)
				continue
			}
		}

		// Override: source value wins
		dst[key] = srcVal
	}

	return dst
}
