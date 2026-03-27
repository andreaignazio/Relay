function(ctx) {
  identity_id: ctx.identity.id,
  email: ctx.identity.traits.email,
  display_name: ctx.identity.traits.display_name,
  username: ctx.identity.traits.username,
}