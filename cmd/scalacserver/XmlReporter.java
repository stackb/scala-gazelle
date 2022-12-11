import java.util.ArrayList;
import java.util.Base64;
import java.util.List;
import javax.print.attribute.standard.Severity;
import scala.reflect.internal.util.Position;
import scala.tools.nsc.reporters.ConsoleReporter;
import scala.tools.nsc.Settings;

public class XmlReporter extends ConsoleReporter {

    final List<Diagnostic> diagnostics = new ArrayList<>();
    final boolean verbose;

    public XmlReporter(Settings settings, boolean verbose) {
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
        diagnostics.add(new Diagnostic(pos, msg, severity));
    }

    static class Diagnostic {
        final Position pos;
        final String msg;
        final Severity sev;

        Diagnostic(final Position pos, final String msg, final Severity sev) {
            this.pos = pos;
            this.msg = msg;
            this.sev = sev;
        }

        @Override
        public String toString() {
            /**
             * Base64 used here as a poor-mans way of avoiding JSON encoding issues with the
             * message.
             */
            return String.format(
                    "{\"pos\": \"%s\", \"sev\": \"%s\", \"msg\": \"%s\"}\n",
                    pos, sev, msg);
        }
    }
}