//go:build windows

package platform

import "os"

func nativeEnvironment() map[string]string {
	result := map[string]string{}
	for _, key := range []string{"SystemRoot", "WINDIR"} {
		if value := os.Getenv(key); value != "" {
			result[key] = value
		}
	}
	return result
}
