package logger

type Log interface {
	// Standard logging methods
	Print(v ...any)
	Printf(format string, v ...any)
	Println(v ...any)

	// Fatal logging methods (log and then call os.Exit(1))
	Fatal(v ...any)
	Fatalf(format string, v ...any)
	Fatalln(v ...any)

	// Panic logging methods (log and then call panic())
	Panic(v ...any)
	Panicf(format string, v ...any)
	Panicln(v ...any)
}
