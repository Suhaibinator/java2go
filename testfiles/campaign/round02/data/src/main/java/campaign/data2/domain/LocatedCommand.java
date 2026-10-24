package campaign.data2.domain;

public final class LocatedCommand {
    public final String source;
    public final int line;
    public final Command command;

    public LocatedCommand(String source, int line, Command command) {
        this.source = source;
        this.line = line;
        this.command = command;
    }

    public String location() { return source + ":" + line; }
}
