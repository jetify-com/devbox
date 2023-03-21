from django.db import models

# Create your models here.
class Todo(models.Model):
    title = models.CharField(max_length=255)
    completed = models.BooleanField(default=False)
    created_att = models.DateTimeField(auto_now_add=True)

    class Meta: 
        app_label = 'todo_app'