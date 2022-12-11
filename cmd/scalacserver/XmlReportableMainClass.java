import scala.tools.nsc.Global;
import scala.tools.nsc.MainClass;
import scala.tools.nsc.Settings;

public class XmlReportableMainClass extends MainClass {
    /**
     * reporter is exposed for convenience.
     */
    public XmlReporter reporter;

    private final boolean verbose;

    public XmlReportableMainClass(boolean verbose) {
        super();
        this.verbose = verbose;
    }

    @Override
    public Global newCompiler() {
        Settings settings = super.settings();
        reporter = new XmlReporter(settings, this.verbose);
        return new Global(settings, reporter);
    }

}