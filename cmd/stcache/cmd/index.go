package cmd

//go:generate sqlboiler sqlite3 -o ../../../pkg/db/sqlite/models/metadata -c ../../../configs/sqlboiler/metadata.yaml
//go:generate go-bindata -pkg metadata -o ../../../pkg/db/sqlite/migrations/metadata/migrations.go ../../../db/sqlite/migrations/metadata

import (
	"archive/tar"
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pojntfx/stfs/pkg/controllers"
	"github.com/pojntfx/stfs/pkg/db/sqlite/migrations/metadata"
	models "github.com/pojntfx/stfs/pkg/db/sqlite/models/metadata"
	"github.com/pojntfx/stfs/pkg/formatting"
	"github.com/pojntfx/stfs/pkg/readers"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

const (
	dbFlag         = "db"
	recordSizeFlag = "record-size"
	recordFlag     = "record"
	blockFlag      = "block"
)

var indexCmd = &cobra.Command{
	Use:     "index",
	Aliases: []string{"i"},
	Short:   "Index contents of tape or tar file",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			return err
		}

		leading, _ := filepath.Split(viper.GetString(dbFlag))
		if err := os.MkdirAll(leading, os.ModePerm); err != nil {
			return err
		}

		db, err := sql.Open("sqlite3", viper.GetString(dbFlag))
		if err != nil {
			return err
		}
		defer db.Close()

		if _, err := migrate.Exec(
			db,
			"sqlite3",
			migrate.AssetMigrationSource{
				Asset:    metadata.Asset,
				AssetDir: metadata.AssetDir,
				Dir:      "../../../db/sqlite/migrations/metadata",
			},
			migrate.Up,
		); err != nil {
			return err
		}

		fileDescription, err := os.Stat(viper.GetString(tapeFlag))
		if err != nil {
			return err
		}

		var f *os.File
		if fileDescription.Mode().IsRegular() {
			f, err = os.Open(viper.GetString(tapeFlag))
			if err != nil {
				return err
			}
		} else {
			f, err = os.OpenFile(viper.GetString(tapeFlag), os.O_RDONLY, os.ModeCharDevice)
			if err != nil {
				return err
			}
		}
		defer f.Close()

		if fileDescription.Mode().IsRegular() {
			// Seek to record and block
			if _, err := f.Seek(int64((viper.GetInt(recordSizeFlag)*controllers.BlockSize*viper.GetInt(recordFlag))+viper.GetInt(blockFlag)*controllers.BlockSize), 0); err != nil {
				return err
			}

			tr := tar.NewReader(f)

			record := viper.GetInt64(recordFlag)
			block := viper.GetInt64(blockFlag)

			for {
				hdr, err := tr.Next()
				if err != nil {
					// Seek right after the next two blocks to skip the trailer
					if _, err := f.Seek((controllers.BlockSize * 2), io.SeekCurrent); err == nil {
						curr, err := f.Seek(0, io.SeekCurrent)
						if err != nil {
							return err
						}

						nextTotalBlocks := curr / controllers.BlockSize
						record = nextTotalBlocks / int64(viper.GetInt(recordSizeFlag))
						block = nextTotalBlocks - (record * int64(viper.GetInt(recordSizeFlag)))

						if block > int64(viper.GetInt(recordSizeFlag)) {
							record++
							block = 0
						}

						tr = tar.NewReader(f)

						hdr, err = tr.Next()
						if err != nil {
							if err == io.EOF {
								break
							}

							return err
						}
					} else {
						return err
					}
				}

				if record == 0 && block == 0 {
					if err := formatting.PrintCSV(formatting.TARHeaderCSV); err != nil {
						return err
					}
				}

				if err := formatting.PrintCSV(formatting.GetTARHeaderAsCSV(record, block, hdr)); err != nil {
					return err
				}

				paxHeaders, err := json.Marshal(hdr.PAXRecords)
				if err != nil {
					return err
				}

				dbhdr := models.Header{
					Record:     record,
					Block:      block,
					Typeflag:   int64(hdr.Typeflag),
					Name:       hdr.Name,
					Linkname:   hdr.Linkname,
					Size:       hdr.Size,
					Mode:       hdr.Mode,
					UID:        int64(hdr.Uid),
					Gid:        int64(hdr.Gid),
					Uname:      hdr.Uname,
					Gname:      hdr.Gname,
					Modtime:    hdr.ModTime,
					Accesstime: hdr.AccessTime,
					Changetime: hdr.ChangeTime,
					Devmajor:   hdr.Devmajor,
					Devminor:   hdr.Devminor,
					Paxrecords: string(paxHeaders),
					Format:     int64(hdr.Format),
				}

				// TODO: Decompose to persister
				if _, err := models.FindHeader(context.Background(), db, dbhdr.Name, models.HeaderColumns.Name); err != nil {
					if err == sql.ErrNoRows {
						if err := dbhdr.Insert(cmd.Context(), db, boil.Infer()); err != nil {
							return err
						}
					} else {
						return err
					}
				} else {
					if _, err := dbhdr.Update(cmd.Context(), db, boil.Infer()); err != nil {
						return err
					}
				}

				curr, err := f.Seek(0, io.SeekCurrent)
				if err != nil {
					return err
				}

				nextTotalBlocks := (curr + hdr.Size) / controllers.BlockSize
				record = nextTotalBlocks / int64(viper.GetInt(recordSizeFlag))
				block = nextTotalBlocks - (record * int64(viper.GetInt(recordSizeFlag)))

				if block > int64(viper.GetInt(recordSizeFlag)) {
					record++
					block = 0
				}
			}
		} else {
			// Seek to record
			if err := controllers.SeekToRecordOnTape(f, int32(viper.GetInt(recordFlag))); err != nil {
				return err
			}

			// Seek to block
			br := bufio.NewReaderSize(f, controllers.BlockSize*viper.GetInt(recordSizeFlag))
			if _, err := br.Read(make([]byte, viper.GetInt(blockFlag)*controllers.BlockSize)); err != nil {
				return err
			}

			record := viper.GetInt64(recordFlag)
			block := viper.GetInt64(blockFlag)

			lastBytesRead := (viper.GetInt(recordSizeFlag) * controllers.BlockSize * viper.GetInt(recordFlag)) + (viper.GetInt(blockFlag) * controllers.BlockSize)
			counter := &readers.Counter{Reader: br, BytesRead: lastBytesRead}
			dirty := false

			for {
				tr := tar.NewReader(counter)
				hdr, err := tr.Next()
				if err != nil {
					if lastBytesRead == counter.BytesRead {
						if dirty {
							// EOD

							break
						}

						if err := controllers.GoToNextFileOnTape(f); err != nil {
							// EOD

							break
						}

						currentRecord, err := controllers.GetCurrentRecordFromTape(f)
						if err != nil {
							return err
						}

						br = bufio.NewReaderSize(f, controllers.BlockSize*viper.GetInt(recordSizeFlag))
						counter = &readers.Counter{Reader: br, BytesRead: (int(currentRecord) * viper.GetInt(recordSizeFlag) * controllers.BlockSize)} // We asume we are at record n, block 0

						dirty = true
					}

					lastBytesRead = counter.BytesRead

					continue
				}

				lastBytesRead = counter.BytesRead

				if hdr.Format == tar.FormatUnknown {
					continue
				}

				dirty = false

				if counter.BytesRead == 0 {
					if err := formatting.PrintCSV(formatting.TARHeaderCSV); err != nil {
						return err
					}
				}

				if err := formatting.PrintCSV(formatting.GetTARHeaderAsCSV(record, block, hdr)); err != nil {
					return err
				}

				nextBytes := int64(counter.BytesRead) + hdr.Size + controllers.BlockSize - 1

				record = nextBytes / (controllers.BlockSize * int64(viper.GetInt(recordSizeFlag)))
				block = (nextBytes - (record * int64(viper.GetInt(recordSizeFlag)) * controllers.BlockSize)) / controllers.BlockSize
			}
		}

		return nil
	},
}

func init() {
	// Get default working dir
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	workingDirDefault := filepath.Join(home, ".local", "share", "stcache", "var", "lib", "stcache")

	indexCmd.PersistentFlags().StringP(dbFlag, "d", filepath.Join(workingDirDefault, "index.sqlite"), "Database to use")
	indexCmd.PersistentFlags().IntP(recordSizeFlag, "e", 20, "Amount of 512-bit blocks per record")
	indexCmd.PersistentFlags().IntP(recordFlag, "r", 0, "Record to seek too before counting")
	indexCmd.PersistentFlags().IntP(blockFlag, "b", 0, "Block in record to seek too before counting")

	viper.AutomaticEnv()

	rootCmd.AddCommand(indexCmd)
}
