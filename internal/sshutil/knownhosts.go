package sshutil

import (
	"bytes"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"golang.org/x/crypto/ssh"
	"net"
)

// knownHostsKeys is a map of host to their public keys in SSH wire format.
//
// This map contains all currently known host keys for the mittwald platform.
// These are very unlikely to change, but if they do, the provider will need to
// be updated accordingly. Also, new clusters may be added in the future.
//
// Alternatively, it would be nice to have an API endpoint to retrieve the
// known host keys, but currently there is no such endpoint available in the
// mittwald API. Therefore, we hardcode the known host keys here.
//
// The format is the same as found in ~/.ssh/known_hosts but without the
// hostname prefix.
var knownHostsKeys = map[string]string{
	"ssh.fabbenstedt.project.host:22": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC3X+n//34lLBZQsxkqgs1BrUdbqpU3KeF9n+hF4XHIiq2tXm7w6mBVWZlebUjFzKyLMDsaz/1AnUajWPp5CjCA1brEGBrCVoC7nwgmsTw5CMN/66Kb/sr3DQLlqGl5ylv+hF9RVmcqKyBbkPIHCGJm1eG+rwEWX0QMNpTyeQxDzxBLTtvcYebgkrxNEo/bs7YvTFoR+yWHt2MqJMnVDbzy/sm0mZCOFgE/jZ2RwyGmPWat63cZIFWucTZ/C73dmwwOyX/RH9kUSaxBm77UTtNpM23dFLDYyh7dw9I0q/7beTNnMplIlo4YX5+mh2288qcFa6v7N/u/KvlzBmkRcRk7",
	"ssh.schmalge.project.host:22":    "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC8DvrhtCe2FQnMPf0caDgTTeyFYMjC/N6xJK9X7OkUgA0A5UPKYQm9/rurJsv9/1TziFYypGF+IMJT3X9/5O9DZMed2hPMmDjEJYYHXZdHT5vSeFf0WuSMWhDEQD757XVA+4ySbyw5+SKsF8HFtuao9moXYMijM+iHp3TWC8be3eJdGkooZundveGIK2xlcgTr3y2LfE9DFMvpx2q4WgDbLrrplvQdJE8eY3Vaz0o251c1PEoOkwCEGQdZxc9XzXHC+SoGV3YPx8WJqhqkM1ayxhOCKJohoN4Nm9H6fcl0tYJCX62+RPi7RRobPzmwbjre8lA2Ibfl3iJVvB830ipj",
	"ssh.vehlage.project.host:22":     "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDTi2M1JMT+yL8F6TVyMvVWpDKQJT9tIAfm32gt5FZoJBMI2NM0REh8eTrI0d5nTiaTQSFskKIXMP6kv1UONaFAWIeh6r5+l3GntyOxcakoE9hqsBzQECXyWGMbav47bkLG3JYaTJchlQeeaV05j53LYvVIjw3mdGQKhnhvyJ7T5TjGCZYU2vnzqCtdXHCXMKAcdfh0XtMLjHBn/GJOolH9jDIzjZo2WbwT5S11tZcOnARFoNxelgIuSHwepz+gbgAILEw8UxjwII0M6kXXM0XgyjL/blOjfbkgiy36qMhDwjsQvA2u336R+UfbONajvBs8heygdVmgPsVc5Uz1PqAD",
	"ssh.gestringen.project.host:22":  "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCv2Ojw4TM0+jFyxUgd9EzAAb8mXIzG24yS4V0Md7yphrl9vgZOyEKFhy8KfXAN1qnT1lMQKwfRqthVnfQbbSBGLSrwF1ecbnNtcXX5eAPIP7ycI+PAJJsLmV2099DpXEh9bOrf+vai29OBwtrq1ukFXatocJbC4nKSPWaFuC6yPcgqTTGMZM3ZxVhl6ONtWyao/IoGgTIHiPGxvKqcmqYYU8Zs6VSnYqQ0Sp+v1S5lgECZsDZf+EvBUe53QW8hzqm7z54ef4Qwhn13ODXU7L75LbfKbe0NJdz5higqaiz9BuIMjfbOsBHSTau5suP5fg/3nWsB3KTjl+P1B4Ipow53",
	"ssh.fiestel.project.host:22":     "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCmLLYqOR4nTxPpUbayAdhf8BxcbjCdzllS4ERhAOb8tmvm29dKrI9RAoFBwHYZL6fh/GnO3lIXpJI0hPtFifsi7WfdtPxK2rk/D8Ijgp19fqhuVQPnDp8WDbPU6dW7xqlWHsPXLIBdfLvPdgkuyC2FBUYG705P6n048DyC8RTJOA57LZ79W5MiaEfxxSqns1l5amhky1qEUmiuW20Y/QJ3aKxliz+Gw6jdSKfM66mAH/3+JhRyB4oGSPoR3IFTjiba6PLhUYzCj1J9lyGxrNAe8vlJeu/wxFjjCqAC0BIyWE4wuxZdjeCjgCKOFIQAuB3ECFg7Qda5tGLPgG58Ni31",
	"ssh.frotheim.project.host:22":    "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDffzlp4b+mFwmVN2aZrm5pwYj/q6VU6P2NKQWsuft3wgroWbo74p2DAqxG904DrcSZJ1ZBbG9up8hBewbMDgMX5pr+A4Nq0nFS7C3+ctrFfpaRXTGOcqxwKlNlrkqhOHDTRvNcZoFd8DseX07YdM5E9vXcRrEFcO4MNuO6jEKtFXE5KRo4SzMUvHrFpDrL608uvv5LTJynkRGu9zrK8AMgURrIGs4GuhsE1/sSYdtu1+r5sMm4tgCrkfxgEi6weJVG4ZCap6tokg/JojMnExNNnhFvNQEg3QiXHMn4vPh45jldezusodhad51mILMPqSmHlDLwuB0KWmfLQidzT+6j",
	"ssh.isenstedt.project.host:22":   "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDA4BkiBXadL4ZCixqcOywUp+l4RnzNEtTTC+Gr+w1dQkXuGg5b6RGJmu+KodFgMyOTPwQMnhj0y0ZQKeHSQVQ4xYLO4kAZNc5AgGPuR9a1cozdLisL8E52fl6YP0ytqOtuH/hsKoIskz1Zl8xUP6mtgVqOT3sZtG29kh3JhngP+JBw94yUs0bOIO84ZPpFbEQ9hmkHMrkHgVoCYpgbV5hnY7tOSyKxWVEQChgXwWe11vpmZzv4XZtnP39bwLbiy4mnOkGqLreXb7kCAljF9hqCOyTaC+mSDdAMsM+qdy7A4SHj6RqCd77QHkmzHJ9gBUnGNX8xMN7+9Rlz3qxK6bqD",
}

// VerifyKnownClusters checks if the provided SSH public key matches the known
// host key for the given hostname.
func VerifyKnownClusters(d *diag.Diagnostics) ssh.HostKeyCallback {
	return func(hostname string, _ net.Addr, key ssh.PublicKey) error {
		// Get the expected key for this host
		expectedKeyStr, exists := knownHostsKeys[hostname]
		if !exists {
			d.AddWarning("Unknown SSH host key", fmt.Sprintf("The host key for %s is not known. The reason for this might be that you are using an outdated version of the mittwald Terraform provider. Please upgrade to a recent version, or open an issue at https://github.com/mittwald/terraform-provider-mittwald if the issue persists.", hostname))
			// Log a warning but do not fail the connection
			// If the host is not in our known hosts, we cannot verify it;
			// accept the host key, in this case.
			return nil
		}

		// Parse the expected key
		expectedKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(expectedKeyStr))
		if err != nil {
			return fmt.Errorf("failed to parse expected host key: %w", err)
		}

		// Compare the keys
		if !bytes.Equal(key.Marshal(), expectedKey.Marshal()) {
			return fmt.Errorf("host key verification failed for %s", hostname)
		}

		return nil
	}
}
