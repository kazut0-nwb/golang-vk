package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Movie struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	Year     int    `json:"year"`
	Director string `json:"director"`
}

type Result struct {
	ID    int
	Movie *Movie
	Err   error
}

func main() {
	fromFlag := flag.Int("from", 0, "id первого фильма")
	toFlag := flag.Int("to", 0, "id последнего фильма")
	workersFlag := flag.Int("workers", 10, "количество воркеров в worker pool")
	timeoutFlag := flag.Duration("timeout", 5*time.Second, "таймаут одного HTTP-запроса")

	flag.Parse()

	fromProvided := false
	toProvided := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "from" {
			fromProvided = true
		}
		if f.Name == "to" {
			toProvided = true
		}
	})

	if !fromProvided || !toProvided {
		fmt.Fprintln(os.Stderr, "Ошибка: флаги --from и --to являются обязательными.")
		flag.Usage()
		os.Exit(1)
	}

	if *fromFlag > *toFlag {
		fmt.Fprintf(os.Stderr, "Ошибка: значение --from (%d) не может быть больше --to (%d)\n", *fromFlag, *toFlag)
		os.Exit(1)
	}

	if *workersFlag <= 0 {
		fmt.Fprintf(os.Stderr, "Ошибка: количество воркеров (--workers) должно быть больше 0, получено: %d\n", *workersFlag)
		os.Exit(1)
	}

	if *fromFlag < 0 {
		fmt.Fprintf(os.Stderr, "Ошибка: значение --from не может быть отрицательным: %d\n", *fromFlag)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	totalTasks := *toFlag - *fromFlag + 1
	jobs := make(chan int, totalTasks)
	results := make(chan Result, totalTasks)

	for id := *fromFlag; id <= *toFlag; id++ {
		jobs <- id
	}
	close(jobs)

	client := &http.Client{}

	var wg sync.WaitGroup
	for w := 1; w <= *workersFlag; w++ {
		wg.Add(1)
		go worker(ctx, &wg, client, jobs, results, *timeoutFlag)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for res := range results {
		if res.Err != nil {
			fmt.Fprintf(os.Stderr, "[Ошибка] фильм ID %d: %v\n", res.ID, res.Err)
			continue
		}
		fmt.Printf("%d — %s — %d — %s\n", res.Movie.ID, res.Movie.Title, res.Movie.Year, res.Movie.Director)
	}
}

func worker(ctx context.Context, wg *sync.WaitGroup, client *http.Client, jobs <-chan int, results chan<- Result, timeout time.Duration) {
	defer wg.Done()

	for id := range jobs {
		select {
		case <-ctx.Done():
			results <- Result{ID: id, Err: ctx.Err()}
			continue
		default:
		}

		movie, err := fetchMovie(ctx, client, id, timeout)
		results <- Result{
			ID:    id,
			Movie: movie,
			Err:   err,
		}
	}
}

func fetchMovie(parentCtx context.Context, client *http.Client, id int, timeout time.Duration) (*Movie, error) {
	reqCtx, cancel := context.WithTimeout(parentCtx, timeout)
	defer cancel()

	url := fmt.Sprintf("https://homeworksite.site/%d/info.0.json", id)
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка сети/таймаут: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("сервер вернул статус %d", resp.StatusCode)
	}

	var movie Movie
	if err := json.NewDecoder(resp.Body).Decode(&movie); err != nil {
		return nil, fmt.Errorf("ошибка декодирования JSON: %w", err)
	}

	if movie.ID == 0 {
		movie.ID = id
	}

	return &movie, nil
}
