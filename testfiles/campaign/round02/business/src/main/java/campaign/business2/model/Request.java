package campaign.business2.model;

public record Request(String id, Customer customer, String sku, int quantity, String route, int priority) {}
