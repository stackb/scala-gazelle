package scalacserver;

import build.stack.gazelle.scala.parse.Diagnostic;

import java.util.ArrayList;
import java.util.Base64;
import java.util.List;
import java.io.File;
import javax.print.attribute.standard.Severity;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import scala.reflect.internal.util.Position;
import scala.tools.nsc.reporters.ConsoleReporter;
import scala.tools.nsc.Settings;

public class DiagnosticReporter extends ConsoleReporter {
    private static final Logger logger = LoggerFactory.getLogger(DiagnosticReporter.class);

    final List<Diagnostic> diagnostics = new ArrayList<>();
    final String dir;

    public DiagnosticReporter(Settings settings, String dir) {
        super(settings);
        this.dir = dir;
    }

    /**
     * @return a copy of the current diagnostics
     */
    public List<Diagnostic> getDiagnostics() {
        return new ArrayList(diagnostics);
    }

    @Override
    public void info0(Position pos, String msg, Severity severity, boolean force) {
        if (logger.isDebugEnabled()) {
            super.info0(pos, msg, severity, force);
        }
        String filename = pos.source().path();
        if (filename != null && filename.startsWith(this.dir)) {
            filename = filename.substring(this.dir.length());
            if (filename.startsWith(File.separator)) {
                filename = filename.substring(1);
            }
        }
        diagnostics.add(Diagnostic.newBuilder()
                .setSeverity(parseSeverity(severity))
                .setSource(filename)
                .setLine(pos.safeLine())
                .setMessage(msg)
                .build());
    }

    private static build.stack.gazelle.scala.parse.Severity parseSeverity(Severity sev) {
        String s = sev.toString();
        if ("ERROR".equals(s)) {
            return build.stack.gazelle.scala.parse.Severity.ERROR;
        }
        if ("WARNING".equals(s)) {
            return build.stack.gazelle.scala.parse.Severity.WARN;
        }
        if ("INFO".equals(s)) {
            return build.stack.gazelle.scala.parse.Severity.INFO;
        }
        logger.warn("failed to parse severity: " + s);
        return build.stack.gazelle.scala.parse.Severity.SEVERITY_UNKNOWN;
    }
}