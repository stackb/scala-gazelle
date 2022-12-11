package scalacserver;

import build.stack.gazelle.scala.parse.Diagnostic;

import java.util.ArrayList;
import java.util.Base64;
import java.util.List;
import javax.print.attribute.standard.Severity;
import scala.reflect.internal.util.Position;
import scala.tools.nsc.reporters.ConsoleReporter;
import scala.tools.nsc.Settings;

public class DiagnosticReporter extends ConsoleReporter {

    final List<Diagnostic> diagnostics = new ArrayList<>();
    final boolean verbose;

    public DiagnosticReporter(Settings settings, boolean verbose) {
        super(settings);
        this.verbose = verbose;
    }

    /**
     * @return a copy of the current diagnostics
     */
    public List<Diagnostic> getDiagnostics() {
        return new ArrayList(diagnostics);
    }

    @Override
    public void info0(Position pos, String msg, Severity severity, boolean force) {
        if (this.verbose) {
            super.info0(pos, msg, severity, force);
        }
        diagnostics.add(Diagnostic.newBuilder()
                .setSeverity(convertSeverity(severity))
                .setSource(pos.source().path())
                .setLine(pos.safeLine())
                .setMessage(msg)
                .build());
    }

    private static build.stack.gazelle.scala.parse.Severity convertSeverity(Severity sev) {
        String s = sev.toString();
        if ("error".equals(s)) {
            return build.stack.gazelle.scala.parse.Severity.ERROR;
        }
        if ("warning".equals(s)) {
            return build.stack.gazelle.scala.parse.Severity.WARN;
        }
        if ("report".equals(s)) {
            return build.stack.gazelle.scala.parse.Severity.INFO;
        }
        return build.stack.gazelle.scala.parse.Severity.SEVERITY_UNKNOWN;
    }
}