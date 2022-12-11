package scalacserver;

import scala.tools.nsc.Global;
import scala.tools.nsc.MainClass;
import scala.tools.nsc.Settings;

public class DiagnosticReportableMainClass extends MainClass {
    /**
     * reporter is exposed for convenience.
     */
    public DiagnosticReporter reporter;

    private final boolean verbose;

    public DiagnosticReportableMainClass(boolean verbose) {
        super();
        this.verbose = verbose;
    }

    @Override
    public Global newCompiler() {
        Settings settings = super.settings();
        reporter = new DiagnosticReporter(settings, this.verbose);
        return new Global(settings, reporter);
    }

}