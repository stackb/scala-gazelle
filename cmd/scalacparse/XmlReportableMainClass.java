package scalaparse;

import scala.tools.nsc.Global;
import scala.tools.nsc.MainClass;
import scala.tools.nsc.Settings;

public class XmlReportableMainClass extends MainClass {
    /**
     * reporter is exposed for convenience.
     */
    public XmlReporter reporter;

    @Override
    public Global newCompiler() {
        Settings settings = super.settings();
        reporter = new XmlReporter(settings);
        return new Global(settings, reporter);
    }

}