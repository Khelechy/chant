package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/khelechy/chant"
	"github.com/khelechy/chant/internal/wavio"
	"github.com/spf13/cobra"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chant",
		Short: "Encode and decode encrypted text as audio",
	}
	cmd.AddCommand(newKeygenCmd())
	cmd.AddCommand(newEncodeCmd())
	cmd.AddCommand(newDecodeCmd())
	return cmd
}

func newKeygenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "keygen",
		Short: "Generate a random 32-byte key as hex",
		RunE: func(cmd *cobra.Command, args []string) error {
			key, err := chant.GenerateKey()
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), key)
			return err
		},
	}
}

func newEncodeCmd() *cobra.Command {
	var keyHex string
	var message string
	var inputPath string
	var outputPath string

	cmd := &cobra.Command{
		Use:   "encode",
		Short: "Encode plaintext into a WAV file",
		RunE: func(cmd *cobra.Command, args []string) error {
			if keyHex == "" {
				return errors.New("chant: --key is required")
			}
			if outputPath == "" {
				return errors.New("chant: --output is required")
			}
			if (message == "" && inputPath == "") || (message != "" && inputPath != "") {
				return errors.New("chant: provide exactly one of --message or --input")
			}

			key, err := chant.KeyFromHex(keyHex)
			if err != nil {
				return err
			}

			plaintext := []byte(message)
			if inputPath != "" {
				plaintext, err = os.ReadFile(inputPath)
				if err != nil {
					return fmt.Errorf("chant: read input file: %w", err)
				}
			}

			samples, err := chant.EncodeMessage(key, plaintext)
			if err != nil {
				return err
			}
			if err := wavio.WriteWAV(outputPath, samples, 48000); err != nil {
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&keyHex, "key", "", "32-byte hex key")
	cmd.Flags().StringVar(&message, "message", "", "Plaintext message to encode")
	cmd.Flags().StringVar(&inputPath, "input", "", "Path to a text file to encode")
	cmd.Flags().StringVar(&outputPath, "output", "", "Output WAV file path")
	return cmd
}

func newDecodeCmd() *cobra.Command {
	var keyHex string
	var inputPath string
	var outputPath string

	cmd := &cobra.Command{
		Use:   "decode",
		Short: "Decode a WAV file back to plaintext",
		RunE: func(cmd *cobra.Command, args []string) error {
			if keyHex == "" {
				return errors.New("chant: --key is required")
			}
			if inputPath == "" {
				return errors.New("chant: --input is required")
			}

			key, err := chant.KeyFromHex(keyHex)
			if err != nil {
				return err
			}

			samples, _, err := wavio.ReadWAV(inputPath)
			if err != nil {
				return err
			}
			plaintext, err := chant.DecodeMessage(key, samples)
			if err != nil {
				return err
			}

			if outputPath == "" {
				_, err = fmt.Fprint(cmd.OutOrStdout(), string(plaintext))
				return err
			}
			if err := os.WriteFile(outputPath, plaintext, 0o644); err != nil {
				return fmt.Errorf("chant: write output file: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&keyHex, "key", "", "32-byte hex key")
	cmd.Flags().StringVar(&inputPath, "input", "", "Input WAV file path")
	cmd.Flags().StringVar(&outputPath, "output", "", "Optional output text file path")
	return cmd
}
