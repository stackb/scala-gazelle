import net.sourceforge.plantuml.SourceStringReader;
import net.sourceforge.plantuml.FileFormat;
import net.sourceforge.plantuml.FileFormatOption;

import java.io.FileOutputStream;
import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Paths;

public class Main {
    public static void main(String[] args) {
        if (args.length != 2) {
            System.err.println("Usage: java PlantUMLToPNG <input.puml> <output.png>");
            System.exit(1);
        }

        String inputFilePath = args[0];
        String outputFilePath = args[1];

        try {
            // Read PlantUML source from file
            String source = new String(Files.readAllBytes(Paths.get(inputFilePath)));

            // Create a SourceStringReader
            SourceStringReader reader = new SourceStringReader(source);

            // Generate the PNG image
            FileOutputStream outputStream = new FileOutputStream(outputFilePath);
            reader.generateImage(outputStream, new FileFormatOption(FileFormat.PNG));
            outputStream.close();

            System.out.println("PNG image generated successfully!");

        } catch (IOException e) {
            System.err.println("Error processing PlantUML file: " + e.getMessage());
            e.printStackTrace();
        }
    }
}