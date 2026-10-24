package prereq.factory.app;

import prereq.factory.api.Adapter;
import prereq.factory.api.Factory;
import prereq.factory.api.Token;
import prereq.factory.cache.Context;
import prereq.factory.domain.Document;
import prereq.factory.domain.Order;
import prereq.factory.impl.DocumentFactory;
import prereq.factory.impl.OrderFactory;

public final class Main {
    private Main() {}

    private static void install(Context context, DocumentFactory<StringBuilder> documents,
                                OrderFactory orders) {
        context.add(new Factory() {
            @Override
            public <T> Adapter<T> create(Context target, Token<T> token) {
                target.probe('X', token.kind());
                return null;
            }
        });
        context.add(documents);
        context.add(orders);
    }

    public static void main(String[] args) {
        int seed = Integer.parseInt(args[0]);
        DocumentFactory<StringBuilder> documents =
                new DocumentFactory<>(new StringBuilder("rev-" + seed));
        OrderFactory orders = new OrderFactory(seed % 5 + 1);
        Token<Document> documentToken = new Token<>("document");
        Token<Order> orderToken = new Token<>("order");
        Context primary = new Context();
        install(primary, documents, orders);

        Adapter<Document> firstDocument = primary.lookup(documentToken);
        Adapter<Order> firstOrder = primary.lookup(orderToken);
        Adapter<Document> repeatedDocument = primary.lookup(documentToken);
        Adapter<Order> repeatedOrder = primary.lookup(orderToken);
        Document document = new Document("doc-" + seed, "draft");
        Order order = new Order("sku-" + seed, seed + 5);
        boolean documentReturnedSelf = firstDocument.apply(document) == document;
        repeatedDocument.apply(document);
        boolean orderReturnedSelf = firstOrder.apply(order) == order;
        repeatedOrder.apply(order);

        Adapter<Document> unknownFirst = primary.lookup(new Token<Document>("unknown"));
        Adapter<Order> unknownSecond = primary.lookup(new Token<Order>("unknown"));
        Adapter<Order> polluted = primary.lookup(new Token<Order>("document"));
        String orderBeforeWrongUse = order.snapshot();
        boolean wrongTypeFailed = false;
        try {
            polluted.apply(order);
        } catch (ClassCastException expected) {
            wrongTypeFailed = true;
        }

        Context secondary = new Context();
        install(secondary, documents, orders);
        Adapter<Document> secondContextDocument = secondary.lookup(documentToken);
        Document otherDocument = new Document("other-" + seed, "new");
        secondContextDocument.apply(otherDocument);

        System.out.println("seed=" + seed + ",identity="
                + (firstDocument == repeatedDocument) + ","
                + (firstOrder == repeatedOrder) + ","
                + (firstDocument != secondContextDocument));
        System.out.println("document=" + document.snapshot() + ",other=" + otherDocument.snapshot()
                + ",returned=" + documentReturnedSelf);
        System.out.println("order=" + order.snapshot() + ",returned=" + orderReturnedSelf
                + ",beforeWrong=" + orderBeforeWrongUse);
        System.out.println("nulls=" + (unknownFirst == null) + "," + (unknownSecond == null)
                + ",wrongType=" + wrongTypeFailed + ",calls=" + firstDocument.calls()
                + "," + firstOrder.calls() + "," + secondContextDocument.calls());
        System.out.println("factories=" + documents.creations() + "," + orders.creations());
        System.out.println("primary=" + primary.snapshot());
        System.out.println("secondary=" + secondary.snapshot());
    }
}
