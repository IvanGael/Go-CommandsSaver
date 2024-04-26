package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/boltdb/bolt"
	"github.com/olekukonko/tablewriter"
)

// Command represents a command with its technology group, reason, and date added.
type Command struct {
	ID         int
	Technology string
	Command    string
	Reason     string
	DateAdded  time.Time
}

func main() {
	// Open the BoltDB database.
	db, err := bolt.Open("commands.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create a bucket for commands if it doesn't exist.
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("commands"))
		return err
	})
	if err != nil {
		log.Fatal(err)
	}

	// Interactive CLI loop
	for {
		fmt.Println("Choose an option:")
		fmt.Println("1. Add a command")
		fmt.Println("2. List all commands")
		fmt.Println("3. Extract commands to file")
		fmt.Println("4. Exit")

		var choice string
		fmt.Print("Enter your choice: ")
		fmt.Scanln(&choice)

		switch choice {
		case "1":
			addCommandInteractive(db)
		case "2":
			listCommands(db)
		case "3":
			extractCommandsToFile(db)
		case "4":
			fmt.Println("Exiting...")
			return
		default:
			fmt.Println("Invalid choice. Please enter a valid option.")
		}
	}
}

// addCommandInteractive adds a new command to the database interactively.
func addCommandInteractive(db *bolt.DB) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Enter the technology:")
	technology, _ := reader.ReadString('\n')
	technology = strings.TrimSpace(technology)

	fmt.Println("Enter the command:")
	command, _ := reader.ReadString('\n')
	command = strings.TrimSpace(command)

	fmt.Println("Enter the reason:")
	reason, _ := reader.ReadString('\n')
	reason = strings.TrimSpace(reason)

	date := time.Now()

	err := addCommand(db, technology, command, reason, date)
	if err != nil {
		log.Println("Error adding command:", err)
		return
	}

	fmt.Println("Command added successfully.")
}

// extractCommandsToFile extracts all commands from the database to a specified text file.
func extractCommandsToFile(db *bolt.DB) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Enter the file path to save the commands (e.g., commands.txt):")
	filePath, _ := reader.ReadString('\n')
	filePath = strings.TrimSpace(filePath)

	file, err := os.Create(filePath)
	if err != nil {
		log.Println("Error creating file:", err)
		return
	}
	defer file.Close()

	commands, err := getCommands(db)
	if err != nil {
		log.Println("Error getting commands:", err)
		return
	}

	for _, cmd := range commands {
		_, err := fmt.Fprintf(file, "ID: %d, Technology: %s, Command: %s, Reason: %s, Date Added: %s\n", cmd.ID, cmd.Technology, cmd.Command, cmd.Reason, cmd.DateAdded.Format("2006-01-02 15:04:05"))
		if err != nil {
			log.Println("Error writing to file:", err)
			return
		}
	}

	fmt.Println("Commands extracted to", filePath, "successfully.")
}

// listCommands retrieves and lists all commands from the database.
func listCommands(db *bolt.DB) {
	commands, err := getCommands(db)
	if err != nil {
		log.Println("Error listing commands:", err)
		return
	}

	if len(commands) == 0 {
		fmt.Println("No commands found.")
		return
	}

	fmt.Println("Commands:")

	// Create a new table
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"ID", "Technology", "Command", "Reason", "Date Added"})

	// Add data to the table
	for _, cmd := range commands {
		table.Append([]string{strconv.Itoa(cmd.ID), cmd.Technology, cmd.Command, cmd.Reason, cmd.DateAdded.Format("2006-01-02 15:04:05")})
	}

	// Render table
	table.Render()
}

// addCommand adds a new command to the database.
func addCommand(db *bolt.DB, technology, command, reason string, date time.Time) error {
	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("commands"))
		id, _ := b.NextSequence()
		cmd := Command{
			ID:         int(id),
			Technology: technology,
			Command:    command,
			Reason:     reason,
			DateAdded:  date,
		}
		encoded, err := encodeCommand(cmd)
		if err != nil {
			return err
		}
		return b.Put(itob(cmd.ID), encoded)
	})
	return err
}

// getCommands retrieves all commands from the database.
func getCommands(db *bolt.DB) ([]Command, error) {
	var commands []Command
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("commands"))
		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			cmd, err := decodeCommand(v)
			if err != nil {
				return err
			}
			commands = append(commands, cmd)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return commands, nil
}

// encodeCommand encodes a command into a byte slice.
func encodeCommand(cmd Command) ([]byte, error) {
	// Simple encoding, you can choose any encoding method here.
	return []byte(fmt.Sprintf("%d,%s,%s,%s,%s", cmd.ID, cmd.Technology, cmd.Command, cmd.Reason, cmd.DateAdded.Format(time.RFC3339))), nil
}

// decodeCommand decodes a byte slice into a command.
func decodeCommand(data []byte) (Command, error) {
	parts := bytes.Split(data, []byte(","))
	id, _ := strconv.Atoi(string(parts[0]))
	date, _ := time.Parse(time.RFC3339, string(parts[4]))
	return Command{
		ID:         id,
		Technology: string(parts[1]),
		Command:    string(parts[2]),
		Reason:     string(parts[3]),
		DateAdded:  date,
	}, nil
}

// itob converts an integer to a byte slice.
func itob(v int) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}
