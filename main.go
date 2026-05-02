package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/duncanawatt-oao/bloggator/internal/config"
	"github.com/duncanawatt-oao/bloggator/internal/database"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type state struct {
	db  *database.Queries
	cfg *config.Config
}

type command struct {
	Name string
	Args []string
}

type commands struct {
	validCommands map[string]func(*state, command) error
}

func (c *commands) run(s *state, cmd command) error {
	toRun, ok := c.validCommands[cmd.Name]
	if !ok {
		return fmt.Errorf("Command %s not found\n", cmd.Name)
	}
	err := toRun(s, cmd)
	if err != nil {
		return err
	}
	return nil
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.validCommands[name] = f
	return
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.Args) == 0 {
		return fmt.Errorf("No username given")
	}
	usr := cmd.Args[0]
	_, err := s.db.GetUser(context.Background(), usr)
	if err != nil {
		return err
	}

	err = s.cfg.SetUser(usr)
	if err != nil {
		return err
	}

	fmt.Printf("User set as %s\n", usr)
	return nil
}

func handlerRegister(s *state, cmd command) error {
	if len(cmd.Args) < 1 {
		return fmt.Errorf("No username given")
	}
	uName := cmd.Args[0]
	usr, err := s.db.CreateUser(context.Background(), database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      uName,
	})
	if err != nil {
		return err
	}
	err = s.cfg.SetUser(usr.Name)
	if err != nil {
		return err
	}
	fmt.Printf("User created: %s\n", usr)
	return nil
}

func handlerReset(s *state, cmd command) error {
	err  := s.db.Reset(context.Background())
	if err == nil {
		fmt.Println("Database reset successfully")
	}
	return err
}

func handlerGetUsers(s *state, cmd command) error {
	usrs, err := s.db.GetUsers(context.Background())
	if err != nil {
		return err 
	}
	for _, usr := range usrs {
		fmt.Printf("* %s", usr.Name)
		if usr.Name == s.cfg.CurrentUserName {
			fmt.Printf(" (current)")
		}
		fmt.Printf("\n")
	}
	return nil
}

func handlerAgg(s *state, cmd command) error {
	feedURL := "https://www.wagslane.dev/index.xml"
	cookie, err := fetchFeed(context.Background(), feedURL)
	if err != nil {
		return err
	}
	fmt.Println(cookie)
	return nil
}

func handlerCreateFeed(s *state, cmd command, user database.User) error {
	if len(cmd.Args) != 2 {
		return fmt.Errorf("Name of feed and URL required")
	}

	name := cmd.Args[0]
	url := cmd.Args[1]

	params := database.CreateFeedParams{
		ID:			uuid.New(),
		CreatedAt:	time.Now(),
		UpdatedAt:	time.Now(),
		Name:		name,
		Url:		url,
		UserID:		user.ID,
	}

	feed, err := s.db.CreateFeed(context.Background(), params)
	if err != nil {
		return err
	}

	fmt.Println("Feed added:")
	fmt.Printf("Feed ID: %v\n", feed.ID)
	fmt.Printf("Created At: %v\n", feed.CreatedAt)
	fmt.Printf("Updated At: %v'\n", feed.UpdatedAt)
	fmt.Printf("Name: %s\n", feed.Name)
	fmt.Printf("URL: %s\n", feed.Url)
	fmt.Printf("User ID: %v\n", feed.UserID)

	followParams := database.CreateFeedFollowParams{
		ID: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		FeedID: feed.ID,
		UserID: feed.UserID,
	}

	_, err = s.db.CreateFeedFollow(context.Background(), followParams)
	if err != nil {
		return err
	}

	fmt.Printf("You are now following %s\n", feed.Name)

	return nil
}

func handlerFeeds(s *state, cmd command) error {
	fds, err := s.db.Feeds(context.Background())
	if err != nil {
		return err 
	}
	for _, feed := range fds {
		fmt.Printf("Feed Name: %s\n", feed.Name)
		fmt.Printf("Feed URL: %s\n", feed.Url)
		fmt.Printf("Username: %s\n", feed.UserName)
		fmt.Println("")
	}

	return nil
}

func handlerFollow(s *state, cmd command, user database.User) error {
	if len(cmd.Args) != 1 {
		return fmt.Errorf("URL required or too many arguments")
	}
	feed, err := s.db.GetFeedByURL(context.Background(), cmd.Args[0])
	if err != nil {
		return err
	}

	params := database.CreateFeedFollowParams{
		ID: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		FeedID: feed.ID,
		UserID: user.ID,
	}
	row, err := s.db.CreateFeedFollow(context.Background(), params)
	if err != nil {
		return err
	}
	fmt.Printf("User %s is now following %s\n", user.Name, row.FeedName)
	return nil
}

func handlerFollowing(s *state, cmd command, user database.User) error {
	feeds, err := s.db.GetFeedFollowsForUser(context.Background(), user.Name)
	if err != nil {
		return err
	}
	fmt.Printf("You (%s) are currently following:\n", user.Name)
	for _, feed := range feeds {
		fmt.Printf("%s\n", feed.FeedName)
	}
	return nil
}

func handlerUnfollow(s *state, cmd command, user database.User) error {
	if len(cmd.Args) != 1 {
		return fmt.Errorf("Command requires URL and URL only as argument")
	}
	params :=  database.UnfollowParams{
		UserID: user.ID,
		Url: cmd.Args[0],
	}
	err := s.db.Unfollow(context.Background(), params)
	if err != nil {
		return err
	}
	fmt.Printf("Unfollowed: %s\n", cmd.Args[0])
	return nil
}

func middlewareLoggedIn(handler func (s *state, cmd command, user database.User) error) func(* state, command) error {
	return func(s *state, cmd command) error {
		usr, err := s.db.GetUser(context.Background(), s.cfg.CurrentUserName)
		if err != nil {
			return err
		}
		return handler(s, cmd, usr)
	}
}


func main() {
	cfg, err := config.Read()
	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("postgres", cfg.DbURL)
	if err != nil {
		log.Fatal(err)
	}

	dbQueries := database.New(db)

	programState := &state{cfg: &cfg, db: dbQueries}

	cmds := commands{
		validCommands: make(map[string]func(*state, command) error),
	}

	cmds.register("login", handlerLogin)
	cmds.register("register", handlerRegister)
	cmds.register("reset", handlerReset)
	cmds.register("users", handlerGetUsers)
	cmds.register("agg", handlerAgg)
	cmds.register("addfeed", middlewareLoggedIn(handlerCreateFeed))
	cmds.register("feeds", handlerFeeds)
	cmds.register("follow", middlewareLoggedIn(handlerFollow))
	cmds.register("following", middlewareLoggedIn(handlerFollowing))
	cmds.register("unfollow", middlewareLoggedIn(handlerUnfollow))

	if len(os.Args) < 2 {
		log.Fatal("No command/arguments given")
	}

	terminalCommand := command{
		Name: os.Args[1],
		Args: os.Args[2:],
	}

	err = cmds.run(programState, terminalCommand)
	if err != nil {
		log.Fatal(err)
	}

}
