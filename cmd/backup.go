package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	backupPkg "github.com/Achilles1089/sovereign-stack/internal/backup"
	"github.com/Achilles1089/sovereign-stack/internal/config"
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Manage encrypted backups",
	Long: `Create, list, and restore encrypted backups of your sovereign data.

Uses Restic for incremental, encrypted, and efficient backups.
All data is encrypted with AES-256 before writing to disk.`,
	RunE: runBackupCreate,
}

var backupListCmd = &cobra.Command{
	Use:   "list",
	Short: "List backup snapshots",
	RunE:  runBackupList,
}

var backupRestoreCmd = &cobra.Command{
	Use:   "restore <snapshot-id>",
	Short: "Restore from a backup snapshot",
	Args:  cobra.ExactArgs(1),
	RunE:  runBackupRestore,
}

var backupPruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Remove old snapshots based on retention policy",
	RunE:  runBackupPrune,
}

var backupInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the backup repository",
	RunE:  runBackupInit,
}

func init() {
	backupCmd.AddCommand(backupListCmd)
	backupCmd.AddCommand(backupRestoreCmd)
	backupCmd.AddCommand(backupPruneCmd)
	backupCmd.AddCommand(backupInitCmd)
	rootCmd.AddCommand(backupCmd)
}

func getBackupManager() *backupPkg.Manager {
	cfgPath := config.ConfigPath(GetConfigPath())
	cfg := config.LoadOrDefault(cfgPath)

	mgr := backupPkg.NewManager(config.ConfigDir())
	if cfg.Backup.Password != "" {
		mgr.SetPassword(cfg.Backup.Password)
	}
	return mgr
}

func runBackupCreate(cmd *cobra.Command, args []string) error {
	fmt.Println()
	fmt.Println("  ⚡ Sovereign Stack — Backup")
	fmt.Println("  ───────────────────────────")
	fmt.Println()

	if !backupPkg.IsResticInstalled() {
		fmt.Println("  ⚠  Restic is not installed.")
		fmt.Println("  Install with:")
		fmt.Println("    macOS:  brew install restic")
		fmt.Println("    Linux:  apt install restic")
		fmt.Println()
		return nil
	}

	mgr := getBackupManager()

	fmt.Println("  Initializing repository (if needed)...")
	if err := mgr.InitRepo(); err != nil {
		return fmt.Errorf("failed to initialize backup repo: %w", err)
	}

	fmt.Println("  Creating encrypted backup snapshot...")
	if err := mgr.Backup("manual"); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	fmt.Println()
	fmt.Println("  ✓ Backup complete!")
	fmt.Println()
	return nil
}

func runBackupList(cmd *cobra.Command, args []string) error {
	fmt.Println()
	fmt.Println("  ⚡ Sovereign Stack — Backup Snapshots")
	fmt.Println("  ──────────────────────────────────────")
	fmt.Println()

	if !backupPkg.IsResticInstalled() {
		return fmt.Errorf("restic is not installed")
	}

	mgr := getBackupManager()
	snapshots, err := mgr.ListSnapshots()
	if err != nil {
		return err
	}

	if len(snapshots) == 0 {
		fmt.Println("  No snapshots found. Create one with: sovereign backup")
		fmt.Println()
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "  ID\tDATE\tHOSTNAME\tTAGS")
	fmt.Fprintln(w, "  ──\t────\t────────\t────")

	for _, snap := range snapshots {
		tags := "-"
		if len(snap.Tags) > 0 {
			tags = snap.Tags[0]
		}
		fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n",
			snap.ID,
			snap.Time.Format("2006-01-02 15:04"),
			snap.Hostname,
			tags,
		)
	}
	w.Flush()

	fmt.Println()
	fmt.Printf("  %d snapshots total\n", len(snapshots))
	fmt.Println()
	return nil
}

func runBackupRestore(cmd *cobra.Command, args []string) error {
	snapshotID := args[0]

	fmt.Println()
	fmt.Printf("  Restoring snapshot %s...\n", snapshotID)

	mgr := getBackupManager()
	if err := mgr.Restore(snapshotID, ""); err != nil {
		return fmt.Errorf("restore failed: %w", err)
	}

	fmt.Println("  ✓ Restore complete!")
	fmt.Println()
	return nil
}

func runBackupPrune(cmd *cobra.Command, args []string) error {
	fmt.Println()
	fmt.Println("  Pruning old snapshots...")
	fmt.Println("  Retention: keep last 7, daily 30, weekly 12")
	fmt.Println()

	mgr := getBackupManager()
	if err := mgr.Prune(7, 30, 12); err != nil {
		return fmt.Errorf("prune failed: %w", err)
	}

	fmt.Println("  ✓ Prune complete!")
	fmt.Println()
	return nil
}

func runBackupInit(cmd *cobra.Command, args []string) error {
	fmt.Println()
	fmt.Println("  Initializing backup repository...")

	mgr := getBackupManager()
	if err := mgr.InitRepo(); err != nil {
		return fmt.Errorf("init failed: %w", err)
	}

	fmt.Println("  ✓ Repository initialized at:", config.ConfigDir()+"/backups")
	fmt.Println()
	return nil
}
