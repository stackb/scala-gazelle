package scalacserver;

import scala.tools.nsc.Global;
import scala.tools.nsc.MainClass;
import scala.tools.nsc.Settings;

public class DiagnosticReportableMainClass extends MainClass {
    private final String dir;
    
    public DiagnosticReporter reporter;

    public DiagnosticReportableMainClass(String dir) {
        super();
        this.dir = dir;
    }

    @Override
    public Global newCompiler() {
        Settings settings = super.settings();
        reporter = new DiagnosticReporter(settings, this.dir);
        return new Global(settings, reporter);
    }
}
